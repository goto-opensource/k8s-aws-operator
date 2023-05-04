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

	"github.com/go-logr/logr"
	awsv1alpha1 "github.com/logmein/k8s-aws-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EIPReconciler reconciles a EIP object
type EIPAssociationReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=aws.k8s.logmein.com,resources=eipassociations,verbs=get;list;watch;create;update;patch;delete

func (r *EIPAssociationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("eipAssociation", req.NamespacedName)

	var eipAssociation awsv1alpha1.EIPAssociation
	if err := r.Get(ctx, req.NamespacedName, &eipAssociation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if eipAssociation.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(eipAssociation.ObjectMeta.Finalizers, finalizerName) {
			eipAssociation.ObjectMeta.Finalizers = append(eipAssociation.ObjectMeta.Finalizers, finalizerName)
			var eip awsv1alpha1.EIP
			if err := r.Client.Get(ctx, client.ObjectKey{
				Namespace: req.Namespace,
				Name:      eipAssociation.Spec.EIPName,
			}, &eip); err != nil {
				return ctrl.Result{}, err
			}
			if eip.Status.State == "allocated" {
				eip.Spec.Assignment = &awsv1alpha1.EIPAssignment{
					PodName: eipAssociation.Spec.PodName,
				}
			}
			if err := r.Update(ctx, &eip); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, r.Update(ctx, &eipAssociation)
		}
	} else {
		// Association is being deleted we want to unassign EIP
		if containsString(eipAssociation.ObjectMeta.Finalizers, finalizerName) {
			var eip awsv1alpha1.EIP
			if err := r.Client.Get(ctx, client.ObjectKey{
				Namespace: req.Namespace,
				Name:      eipAssociation.Spec.EIPName,
			}, &eip); err != nil {
				return ctrl.Result{}, err
			}

			if eip.Status.Assignment != nil && eip.Status.Assignment.PodName == eipAssociation.Spec.PodName {
				log.Info("Unassigning corresponding EIP")
				eip.Status.Assignment = nil
				eip.Spec.Assignment = nil
				if err := r.Update(ctx, &eip); err != nil {
					return ctrl.Result{}, err
				}
			}
			eipAssociation.ObjectMeta.Finalizers = removeString(eipAssociation.ObjectMeta.Finalizers, finalizerName)
			return ctrl.Result{}, r.Update(ctx, &eipAssociation)
		}
	}

	return ctrl.Result{}, nil
}

func (r *EIPAssociationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.EIPAssociation{}).
		Complete(r)
}
