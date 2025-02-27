/*
Copyright 2024.

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

package accesslist

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/internal/controller"
)

const (
	accessListFinalizer = "accesslist.nginxpm-operator.io/finalizers"

	ACL_TOKEN_FIELD = ".spec.token.name"
)

// AccessListReconciler reconciles a AccessList object
type AccessListReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=accesslists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=accesslists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=accesslists/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the AccessList object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *AccessListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	acl := &nginxpmoperatoriov1.AccessList{}

	err := r.Get(ctx, req.NamespacedName, acl)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("accessList resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get accessList")
		return ctrl.Result{}, err
	}

	isMarkedToBeDeleted := !acl.ObjectMeta.DeletionTimestamp.IsZero()

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	if !isMarkedToBeDeleted {
		if err := controller.AddFinalizer(r, ctx, accessListFinalizer, acl); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Let's just set the status as Unknown when no status is available
	if len(acl.Status.Conditions) == 0 {
		controller.UpdateStatus(ctx, r.Client, acl, req.NamespacedName, func() {
			meta.SetStatusCondition(&acl.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionUnknown,
				Type:               controller.ConditionTypeReconciling,
				Reason:             "Reconciling",
				Message:            "Starting reconciliation",
				LastTransitionTime: metav1.Now(),
			})
		})
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccessListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.AccessList{},

		ACL_TOKEN_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			acl := rawObj.(*nginxpmoperatoriov1.AccessList)

			if acl.Spec.Token == nil {
				// If token is not provided, use the default token name
				return []string{controller.TOKEN_DEFAULT_NAME}
			}

			if acl.Spec.Token.Name == "" {
				return nil
			}

			return []string{acl.Spec.Token.Name}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.AccessList{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Named("accesslist").
		Complete(r)
}
