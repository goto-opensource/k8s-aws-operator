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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	awsv1alpha1 "github.com/logmein/k8s-eip-controller/api/v1alpha1"
)

const (
	finalizerName = "eip-release.aws.k8s.logmein.com"
)

// EIPReconciler reconciles a EIP object
type EIPReconciler struct {
	client.Client

	Log logr.Logger

	ec2 *ec2.EC2
}

// +kubebuilder:rbac:groups=aws.k8s.logmein.com,resources=eips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws.k8s.logmein.com,resources=eips/status,verbs=get;update;patch

func (r *EIPReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("eip", req.NamespacedName)

	var eip awsv1alpha1.EIP
	if err := r.Get(ctx, req.NamespacedName, &eip); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := &eip.Status
	spec := &eip.Spec

	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(eip.ObjectMeta.Finalizers, finalizerName) {
			// add finalizer, set initial state
			eip.ObjectMeta.Finalizers = append(eip.ObjectMeta.Finalizers, finalizerName)
			status.State = "allocating"
			return ctrl.Result{}, r.Update(context.Background(), &eip)
		}

		if status.State == "allocating" {
			return ctrl.Result{}, r.allocateEIP(&eip, log)
		}

		resp, err := r.ec2.DescribeAddresses(&ec2.DescribeAddressesInput{
			AllocationIds: []*string{aws.String(status.AllocationId)},
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidAllocationID.NotFound" {
				log.Info("allocation ID not found; assuming EIP was released; not doing anything", "allocationId", eip.Status.AllocationId)
			}
			return ctrl.Result{}, err
		}

		addr := resp.Addresses[0]

		if err := r.reconcileTags(&eip, addr.Tags); err != nil {
			return ctrl.Result{}, err
		}

		if status.State == "allocated" {
			if spec.Assignment != nil {
				if spec.Assignment.PodName != "" {
					status.State = "assigning"
					return ctrl.Result{}, r.Update(context.Background(), &eip)
				}
			}
		}

		if status.State == "assigned" {
			changed := false
			if spec.Assignment == nil {
				// assignment was removed
				status.State = "unassigning"
				changed = true
			} else if *(spec.Assignment) != *(status.Assignment) || addr.AssociationId == nil {
				// assignment was changed (in spec or in EC2)
				status.State = "reassigning"
				changed = true
			}

			if changed {
				return ctrl.Result{}, r.Update(context.Background(), &eip)
			}
		}

		if status.State == "assigning" {
			if spec.Assignment == nil {
				// assignment was removed before EIP was actually assigned
				status.State = "allocated"
				return ctrl.Result{}, r.Update(context.Background(), &eip)
			}
		}

		if status.State == "assigning" || status.State == "reassigning" {
			if spec.Assignment != nil {
				if spec.Assignment.PodName != "" {
					return ctrl.Result{}, r.assignEIPToPod(&eip, log)
				}
			}
		}

		if status.State == "unassigning" {
			return ctrl.Result{}, r.unassignEIP(&eip, log)
		}
	} else {
		// EIP object is being deleted
		if containsString(eip.ObjectMeta.Finalizers, finalizerName) {
			if status.State == "assigned" || status.State == "reassigning" {
				status.State = "unassigning"
				return ctrl.Result{}, r.Update(context.Background(), &eip)
			}

			if status.State == "unassigning" {
				return ctrl.Result{}, r.unassignEIP(&eip, log)
			}

			if status.State == "allocated" {
				status.State = "releasing"
				return ctrl.Result{}, r.Update(context.Background(), &eip)
			}

			if status.State == "releasing" {
				if err := r.releaseEIP(&eip, log); err != nil {
					return ctrl.Result{}, err
				}
			}

			// remove finalizer, allow k8s to remove the resource
			eip.ObjectMeta.Finalizers = removeString(eip.ObjectMeta.Finalizers, finalizerName)
			return ctrl.Result{}, r.Update(context.Background(), &eip)
		}
	}

	return ctrl.Result{}, nil
}

func (r *EIPReconciler) allocateEIP(eip *awsv1alpha1.EIP, log logr.Logger) error {
	log.Info("allocating")

	input := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}
	if eip.Spec.PublicIPAddress != "" {
		input.Address = aws.String(eip.Spec.PublicIPAddress)
	} else if eip.Spec.PublicIPv4Pool != "" {
		input.PublicIpv4Pool = aws.String(eip.Spec.PublicIPv4Pool)
	}

	if resp, err := r.ec2.AllocateAddress(input); err != nil {
		return err
	} else {
		eip.Status.State = "allocated"
		eip.Status.AllocationId = aws.StringValue(resp.AllocationId)
		eip.Status.PublicIPAddress = aws.StringValue(resp.PublicIp)
		r.Log.Info("allocated", "allocationId", eip.Status.AllocationId)
		if err := r.Update(context.Background(), eip); err != nil {
			return err
		}
	}

	return r.reconcileTags(eip, []*ec2.Tag{})
}

func (r *EIPReconciler) reconcileTags(eip *awsv1alpha1.EIP, existingTags []*ec2.Tag) error {
	if eip.Spec.Tags == nil {
		return nil
	}

	resources := []*string{aws.String(eip.Status.AllocationId)}

	var tagsToCreate []*ec2.Tag
	for k, v := range *eip.Spec.Tags {
		create := true
		for _, tag := range existingTags {
			if aws.StringValue(tag.Key) == k && aws.StringValue(tag.Value) == v {
				create = false
				break
			}
		}
		if create {
			tagsToCreate = append(tagsToCreate, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
		}
	}
	if len(tagsToCreate) > 0 {
		if _, err := r.ec2.CreateTags(&ec2.CreateTagsInput{
			Resources: resources,
			Tags:      tagsToCreate,
		}); err != nil {
			return err
		}
	}

	var tagsToRemove []*ec2.Tag
	for _, tag := range existingTags {
		if _, ok := (*eip.Spec.Tags)[aws.StringValue(tag.Key)]; !ok {
			tagsToRemove = append(tagsToRemove, tag)
		}
	}
	if len(tagsToRemove) > 0 {
		_, err := r.ec2.DeleteTags(&ec2.DeleteTagsInput{
			Resources: resources,
			Tags:      tagsToRemove,
		})
		return err
	}

	return nil
}

func (r *EIPReconciler) releaseEIP(eip *awsv1alpha1.EIP, log logr.Logger) error {
	log.Info("releasing")

	if _, err := r.ec2.ReleaseAddress(&ec2.ReleaseAddressInput{
		AllocationId: aws.String(eip.Status.AllocationId),
	}); err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "InvalidAllocationID.NotFound" {
			log.Info("allocation ID not found; assuming EIP already released", "allocationId", eip.Status.AllocationId)
		} else {
			return err
		}
	}

	log.Info("released")

	return nil
}

func (r *EIPReconciler) getPodPrivateIP(namespace, podName string) (string, error) {
	pod := &corev1.Pod{}
	if err := r.Client.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      podName,
	}, pod); err != nil {
		return "", err
	}

	return pod.Status.PodIP, nil
}

func (r *EIPReconciler) findENI(privateIP string) (string, error) {
	if resp, err := r.ec2.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("addresses.private-ip-address"),
				Values: []*string{
					aws.String(privateIP),
				},
			},
		},
	}); err != nil {
		return "", err
	} else {
		if len(resp.NetworkInterfaces) == 0 {
			return "", errors.New("No ENI with private IP of pod found")
		}

		return aws.StringValue(resp.NetworkInterfaces[0].NetworkInterfaceId), nil
	}
}

func (r *EIPReconciler) assignEIPToPod(eip *awsv1alpha1.EIP, log logr.Logger) error {
	privateIP, err := r.getPodPrivateIP(eip.Namespace, eip.Spec.Assignment.PodName)
	if err != nil {
		return err
	}

	if privateIP == "" {
		return errors.New("Pod has no IP")
	}

	eni, err := r.findENI(privateIP)
	if err != nil {
		return err
	}

	log.Info("assigning", "podName", eip.Spec.Assignment.PodName, "privateIP", privateIP, "eni", eni)

	resp, err := r.ec2.AssociateAddress(&ec2.AssociateAddressInput{
		AllowReassociation: aws.Bool(eip.Status.State == "reassigning"),
		AllocationId:       aws.String(eip.Status.AllocationId),
		NetworkInterfaceId: aws.String(eni),
		PrivateIpAddress:   aws.String(privateIP),
	})
	if err != nil {
		return err
	}

	log.Info("assigned")

	eip.Status.State = "assigned"
	eip.Status.AssociationId = aws.StringValue(resp.AssociationId)
	eip.Status.Assignment = eip.Spec.Assignment
	if err := r.Update(context.Background(), eip); err != nil {
		return err
	}

	return nil
}

func (r *EIPReconciler) unassignEIP(eip *awsv1alpha1.EIP, log logr.Logger) error {
	log.Info("unassigning")

	if _, err := r.ec2.DisassociateAddress(&ec2.DisassociateAddressInput{
		AssociationId: aws.String(eip.Status.AssociationId),
	}); err != nil {
		return err
	}

	log.Info("unassigned")

	eip.Status.State = "allocated"
	eip.Status.Assignment = nil
	if err := r.Update(context.Background(), eip); err != nil {
		return err
	}

	return nil
}

func (r *EIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	sess := session.Must(session.NewSession())

	r.ec2 = ec2.New(sess)

	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.EIP{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
