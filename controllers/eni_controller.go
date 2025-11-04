/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"

	//"strings"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	awsv1alpha1 "github.com/logmein/k8s-aws-operator/api/v1alpha1"
)

// ENIReconciler reconciles a ENI object
type ENIReconciler struct {
	client.Client
	NonCachingClient client.Client
	Log              logr.Logger
	EC2              *ec2.EC2
	Tags             map[string]string
}

// +kubebuilder:rbac:groups=aws.k8s.logmein.com,resources=enis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws.k8s.logmein.com,resources=enis/status,verbs=get;update;patch

func (r *ENIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//log := r.Log.WithValues("eni", req.NamespacedName)

	var eni awsv1alpha1.ENI
	if err := r.Get(ctx, req.NamespacedName, &eni); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if eni.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(eni.ObjectMeta.Finalizers, finalizerName) {
			eni.ObjectMeta.Finalizers = append(eni.ObjectMeta.Finalizers, finalizerName)
			return ctrl.Result{}, r.Update(context.Background(), &eni)
		}

		securityGroupIDs, err := r.getSecurityGroupIDs(eni.Spec.SecurityGroups)
		if err != nil {
			return ctrl.Result{}, err
		}

		if eni.Status.NetworkInterfaceID == "" {
			input := &ec2.CreateNetworkInterfaceInput{
				SubnetId:    aws.String(eni.Spec.SubnetID),
				Groups:      securityGroupIDs,
				Description: aws.String(eni.Spec.Description),
			}
			if eni.Spec.SecondaryPrivateIPAddressCount > 0 {
				input.SecondaryPrivateIpAddressCount = aws.Int64(eni.Spec.SecondaryPrivateIPAddressCount)
			}

			tags := ec2.TagSpecification{
				ResourceType: aws.String("elastic-ip"),
			}
			for k, v := range r.Tags {
				tags.Tags = append(tags.Tags, &ec2.Tag{
					Key:   aws.String(k),
					Value: aws.String(v),
				})
			}
			input.TagSpecifications = []*ec2.TagSpecification{&tags}

			resp, err := r.EC2.CreateNetworkInterface(input)
			if err != nil {
				return ctrl.Result{}, err
			}
			eni.Status.NetworkInterfaceID = aws.StringValue(resp.NetworkInterface.NetworkInterfaceId)
			eni.Status.MacAddress = aws.StringValue(resp.NetworkInterface.MacAddress)
			eni.Status.PrivateIPAddresses = r.getPrivateIPAddresses(resp.NetworkInterface.PrivateIpAddresses)
			if err := r.Update(ctx, &eni); err != nil {
				return ctrl.Result{}, err
			}
			if int64(len(eni.Status.PrivateIPAddresses)) != 1+eni.Spec.SecondaryPrivateIPAddressCount {
				return ctrl.Result{
					RequeueAfter: 5 * time.Second,
				}, nil
			}
			return ctrl.Result{}, nil
		}

		resp, err := r.EC2.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []*string{aws.String(eni.Status.NetworkInterfaceID)},
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		eniInfo := resp.NetworkInterfaces[0]

		// reconcile description and security groups
		if aws.StringValue(eniInfo.Description) != eni.Spec.Description {
			_, err = r.EC2.ModifyNetworkInterfaceAttribute(&ec2.ModifyNetworkInterfaceAttributeInput{
				NetworkInterfaceId: aws.String(eni.Status.NetworkInterfaceID),
				Description:        &ec2.AttributeValue{Value: aws.String(eni.Spec.Description)},
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		modify := len(eniInfo.Groups) != len(eni.Spec.SecurityGroups)
		if !modify {
			for _, g := range eniInfo.Groups {
				found := false
				for _, sg := range securityGroupIDs {
					if aws.StringValue(sg) == aws.StringValue(g.GroupId) {
						found = true
						break
					}
				}
				if !found {
					modify = true
					break
				}
			}
		}
		if modify {
			_, err = r.EC2.ModifyNetworkInterfaceAttribute(&ec2.ModifyNetworkInterfaceAttributeInput{
				NetworkInterfaceId: aws.String(eni.Status.NetworkInterfaceID),
				Groups:             securityGroupIDs,
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// reconcile secondary IP address count
		actualNum := int64(len(eniInfo.PrivateIpAddresses))
		desiredNum := 1 + eni.Spec.SecondaryPrivateIPAddressCount
		if actualNum != desiredNum {
			if actualNum < desiredNum {
				_, err = r.EC2.AssignPrivateIpAddresses(&ec2.AssignPrivateIpAddressesInput{
					NetworkInterfaceId:             aws.String(eni.Status.NetworkInterfaceID),
					SecondaryPrivateIpAddressCount: aws.Int64(desiredNum - actualNum),
				})
			} else {
				addressesToRemove := []*string{}
				for _, address := range eniInfo.PrivateIpAddresses[desiredNum:] {
					addressesToRemove = append(addressesToRemove, address.PrivateIpAddress)
				}
				_, err = r.EC2.UnassignPrivateIpAddresses(&ec2.UnassignPrivateIpAddressesInput{
					NetworkInterfaceId: aws.String(eni.Status.NetworkInterfaceID),
					PrivateIpAddresses: addressesToRemove,
				})
			}
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{
				RequeueAfter: 5 * time.Second,
			}, nil
		}
		if len(eni.Status.PrivateIPAddresses) != len(eniInfo.PrivateIpAddresses) {
			eni.Status.PrivateIPAddresses = r.getPrivateIPAddresses(eniInfo.PrivateIpAddresses)
			return ctrl.Result{}, r.Update(ctx, &eni)
		}

		// reconcile pod attachment
		if eni.Spec.Attachment == nil {
			if eniInfo.Attachment == nil || aws.StringValue(eniInfo.Attachment.Status) != "attached" {
				return ctrl.Result{}, nil
			} else {
				err = r.detachENI(aws.StringValue(eniInfo.Attachment.AttachmentId))
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			desiredInstanceID, err := r.getInstanceIDOfPod(eni.Namespace, eni.Spec.Attachment.PodName)
			if err != nil {
				return ctrl.Result{}, err
			}
			if eniInfo.Attachment == nil {
				err = r.attachENI(eni.Status.NetworkInterfaceID, desiredInstanceID)
				if err != nil {
					return ctrl.Result{}, err
				}
			} else {
				if desiredInstanceID == aws.StringValue(eniInfo.Attachment.InstanceId) {
					return ctrl.Result{}, nil
				}
				err = r.detachENI(aws.StringValue(eniInfo.Attachment.AttachmentId))
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidAttachmentID.NotFound" {
						err = nil
					}
				}
				return ctrl.Result{RequeueAfter: 3 * time.Second}, err
			}
		}
		eni.Status.Attachment = eni.Spec.Attachment
		return ctrl.Result{}, r.Update(ctx, &eni)
	} else if containsString(eni.ObjectMeta.Finalizers, finalizerName) {
		if eni.Status.NetworkInterfaceID != "" {
			resp, err := r.EC2.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
				NetworkInterfaceIds: []*string{aws.String(eni.Status.NetworkInterfaceID)},
			})
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "InvalidNetworkInterfaceID.NotFound" {
					return ctrl.Result{}, err
				}
			} else {
				eniInfo := resp.NetworkInterfaces[0]
				if eniInfo.Attachment != nil && aws.StringValue(eniInfo.Attachment.Status) == "attached" {
					err := r.detachENI(aws.StringValue(eniInfo.Attachment.AttachmentId))
					if err != nil {
						if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "InvalidAttachmentID.NotFound" {
							return ctrl.Result{}, err
						}
					}
					eni.Status.Attachment = nil
					return ctrl.Result{}, r.Update(ctx, &eni)
				}
				_, err = r.EC2.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
					NetworkInterfaceId: aws.String(eni.Status.NetworkInterfaceID),
				})
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != "InvalidNetworkInterfaceID.NotFound" {
						return ctrl.Result{}, err
					}
				}
			}
		}
		eni.Status = awsv1alpha1.ENIStatus{}
		eni.ObjectMeta.Finalizers = removeString(eni.ObjectMeta.Finalizers, finalizerName)
		return ctrl.Result{}, r.Update(ctx, &eni)
	}

	return ctrl.Result{}, nil
}

func (r *ENIReconciler) getPrivateIPAddresses(privateIPAddresses []*ec2.NetworkInterfacePrivateIpAddress) []string {
	ret := []string{}
	for _, ip := range privateIPAddresses {
		ret = append(ret, aws.StringValue(ip.PrivateIpAddress))
	}
	return ret
}

func (r *ENIReconciler) getSecurityGroupIDs(securityGroups []string) ([]*string, error) {
	/*
		ret := []*string{}
		descs := []*string{}
		for _, sg := range securityGroups {
			if strings.HasPrefix(sg, "sg-") {
				ret = append(ret, aws.String(sg))
			} else {
				descs = append(ret, aws.String(sg))
			}
		}
		resp, err := r.ec2.DescribeSecurityGroups(&ec2.DescribeSecurityGroups{
			Filters: []*ec2.Filter{
				Name: aws.String("description"),
				Values: descs,
			},
		})
		if err != nil {
			return ret, nil
		}
		for _, sg := range resp.SecurityGroups {
			ret = append(ret, sg.GroupId)
		}
		return ret, nil
	*/
	return aws.StringSlice(securityGroups), nil
}

func (r *ENIReconciler) getPodPrivateIP(namespace, podName string) (string, error) {
	pod := &corev1.Pod{}
	// we use a non-caching client here as otherwise we would need to cache all pods (would increase memory usage) in the cluster and require list/watch permissions
	if err := r.NonCachingClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      podName,
	}, pod); err != nil {
		return "", err
	}

	return pod.Status.PodIP, nil
}

func (r *ENIReconciler) findENI(privateIP string) (*ec2.NetworkInterface, error) {
	if resp, err := r.EC2.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("addresses.private-ip-address"),
				Values: []*string{
					aws.String(privateIP),
				},
			},
		},
	}); err != nil {
		return nil, err
	} else {
		if len(resp.NetworkInterfaces) == 0 {
			return nil, errors.New("No ENI with private IP of pod found")
		}

		return resp.NetworkInterfaces[0], nil
	}
}

func (r *ENIReconciler) getInstanceIDOfPod(namespace, podName string) (string, error) {
	privateIP, err := r.getPodPrivateIP(namespace, podName)
	if err != nil {
		return "", err
	}

	eniInfo, err := r.findENI(privateIP)
	if err != nil {
		return "", err
	}
	if eniInfo.Attachment == nil || aws.StringValue(eniInfo.Attachment.Status) != "attached" {
		return "", errors.New("ENI corresponding to pod IP is not attached")
	}

	return aws.StringValue(eniInfo.Attachment.InstanceId), nil
}

func (r *ENIReconciler) attachENI(attachmentID, instanceID string) error {
	resp, err := r.EC2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return err
	}

	_, err = r.EC2.AttachNetworkInterface(&ec2.AttachNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(attachmentID),
		InstanceId:         aws.String(instanceID),
		DeviceIndex:        aws.Int64(int64(len(resp.Reservations[0].Instances[0].NetworkInterfaces))),
	})
	return err
}

func (r *ENIReconciler) detachENI(attachmentID string) error {
	_, err := r.EC2.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{
		AttachmentId: aws.String(attachmentID),
	})
	return err
}

func (r *ENIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.ENI{}).
		Complete(r)
}
