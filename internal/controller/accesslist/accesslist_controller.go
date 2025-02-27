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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/internal/controller"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
)

const (
	accessListFinalizer = "accesslist.nginxpm-operator.io/finalizers"

	ACL_TOKEN_FIELD = ".spec.token.name"
)

// AccessListReconciler reconciles a AccessList object
type AccessListReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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

	// Create a new Nginx Proxy Manager client
	nginxpmClient, err := controller.InitNginxPMClient(ctx, r, req, acl.Spec.Token)
	if err != nil {
		if isMarkedToBeDeleted {
			// Remove the finalizer
			if err := controller.RemoveFinalizer(r, ctx, accessListFinalizer, acl); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		r.Recorder.Event(
			acl, "Warning", "InitNginxPMClient",
			fmt.Sprintf("Failed to init nginxpm client: ResourceName: %s, Namespace: %s, err: %s",
				req.Name, req.Namespace, err.Error()),
		)

		// Set the status as False when the client can't be created
		controller.UpdateStatus(ctx, r.Client, acl, req.NamespacedName, func() {
			meta.SetStatusCondition(&acl.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionFalse,
				Type:               controller.ConditionTypeError,
				Reason:             "InitNginxPMClient",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
		})

		return ctrl.Result{}, err
	}

	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(acl, accessListFinalizer) {
			log.Info("Performing Finalizer Operations for AccessList")

			if acl.Status.Id != nil {
				// Delete access list here
				err := nginxpmClient.DeleteAccessList(int(*acl.Status.Id))
				if err != nil {
					log.Error(err, "Failed to delete access from remote NPM")
				}
			}

			// Remove the finalizer
			if err := controller.RemoveFinalizer(r, ctx, accessListFinalizer, acl); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Create or update access list
	err = r.createOrUpdateAccessList(ctx, req, acl, nginxpmClient)
	if err != nil {
		// Set the status as False when the client can't be created
		controller.UpdateStatus(ctx, r.Client, acl, req.NamespacedName, func() {
			meta.SetStatusCondition(&acl.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionFalse,
				Type:               controller.ConditionTypeError,
				Reason:             "createOrUpdateAccessList",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
		})

		return ctrl.Result{}, err
	}

	// Set the status as True when the client can be created
	controller.UpdateStatus(ctx, r.Client, acl, req.NamespacedName, func() {
		meta.SetStatusCondition(&acl.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionTrue,
			Type:               controller.ConditionTypeReady,
			Reason:             "createOrUpdateAccessList",
			Message:            fmt.Sprintf("access list created or updated, ResourceName: %s", req.Name),
			LastTransitionTime: metav1.Now(),
		})
	})

	return ctrl.Result{}, nil
}

func (r *AccessListReconciler) createOrUpdateAccessList(ctx context.Context, req ctrl.Request, acl *nginxpmoperatoriov1.AccessList, nginxpmClient *nginxpm.Client) error {
	log := log.FromContext(ctx)

	var accessList *nginxpm.AccessList
	var err error

	if acl.Status.Id != nil {
		accessList, err = nginxpmClient.FindAccessListByID(*acl.Status.Id)
		if err != nil {
			r.Recorder.Event(
				acl, "Warning", "FindAccessListByID",
				fmt.Sprintf("Failed to find access list by ID, ResourceName: %s, Namespace: %s, err: %s",
					req.Name, req.Namespace, err.Error()),
			)

			log.Error(err, "Failed to find access list by ID")
			return err
		}
	}

	authorizations := make([]nginxpm.AccessListItem, len(acl.Spec.Authorizations))
	for i, authorization := range acl.Spec.Authorizations {
		authorizations[i] = nginxpm.AccessListItem{
			Username: authorization.Username,
			Password: authorization.Password,
		}
	}

	clients := make([]nginxpm.AccessListClient, len(acl.Spec.Clients))
	for i, client := range acl.Spec.Clients {
		clients[i] = nginxpm.AccessListClient{
			Address:   client.Address,
			Directive: client.Directive,
		}
	}

	input := nginxpm.AccessListRequestInput{
		Name:       acl.Name,
		SatisfyAny: acl.Spec.SatisfyAny,
		PassAuth:   acl.Spec.PassAuth,
		Items:      authorizations,
		Clients:    clients,
	}

	if accessList == nil {
		accessList, err = nginxpmClient.CreateAccessList(input)
		if err != nil {
			r.Recorder.Event(
				acl, "Warning", "CreateAccessList",
				fmt.Sprintf("Failed to create access list, ResourceName: %s, Namespace: %s, err: %s",
					req.Name, req.Namespace, err.Error()),
			)

			log.Error(err, "Failed to create access list")
			return err
		}

		log.Info("AccessList created successfully")
	} else {
		accessList, err = nginxpmClient.UpdateAccessList(accessList.ID, input)
		if err != nil {
			r.Recorder.Event(
				acl, "Warning", "UpdateAccessList",
				fmt.Sprintf("Failed to update access list, ResourceName: %s, Namespace: %s, err: %s",
					req.Name, req.Namespace, err.Error()),
			)

			log.Error(err, "Failed to update access list")
			return err
		}

		log.Info("AccessList updated successfully")
	}

	return controller.UpdateStatus(ctx, r.Client, acl, req.NamespacedName, func() {
		acl.Status.Id = &accessList.ID
		acl.Status.ProxyHostCount = accessList.ProxyHostCount
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccessListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.AccessList{},

		ACL_TOKEN_FIELD,

		func(rawObj client.Object) []string {
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
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(ACL_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("accesslist").
		Complete(r)
}

func (r *AccessListReconciler) findObjectsForMap(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.AccessListList{}

		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, object.GetName()),
		}

		if field != ACL_TOKEN_FIELD {
			listOps.Namespace = object.GetNamespace()
		}

		err := r.List(ctx, attachedObjects, listOps)
		if err != nil {
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, len(attachedObjects.Items))
		for i, item := range attachedObjects.Items {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			}
		}

		return requests
	}
}
