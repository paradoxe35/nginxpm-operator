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

package controller

import (
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
)

const (
	proxyHostFinalizer = "proxyhost.nginxpm-operator.io/finalizers"

	PH_TOKEN_FIELD = ".spec.token.name"

	PH_CUSTOM_CERTIFICATE_FIELD = ".spec.ssl.customCertificate.name"

	PH_LETSENCRYPT_CERTIFICATE_FIELD = ".spec.ssl.letsEncryptCertificate.name"

	PH_FORWARD_SERVICE_FIELD = ".spec.forward.service.name"

	PH_CUSTOM_LOCATION_FORWARD_FIELD = ".spec.customLocations.forward.service.name"
)

// ProxyHostReconciler reconciles a ProxyHost object
type ProxyHostReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=proxyhosts/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ProxyHost object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ProxyHostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	ph := &nginxpmoperatoriov1.ProxyHost{}

	// Fetch the ProxyHost instance
	err := r.Get(ctx, req.NamespacedName, ph)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("proxyHost resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get proxyHost")
		return ctrl.Result{}, err
	}

	isMarkedToBeDeleted := !ph.ObjectMeta.DeletionTimestamp.IsZero()

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	if !isMarkedToBeDeleted {
		if err := AddFinalizer(r, ctx, proxyHostFinalizer, ph); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create a new Nginx Proxy Manager client
	nginxpmClient, err := InitNginxPMClient(ctx, r, ph.Spec.Token.Name, ph.Spec.Token.Namespace)
	if err != nil {
		// Stop reconciliation if the resource is marked for deletion and the client can't be created
		if isMarkedToBeDeleted {
			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, proxyHostFinalizer, ph); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		} else {
			r.Recorder.Event(
				ph, "Warning", "InitNginxPMClient",
				fmt.Sprintf("Failed to init nginxpm client, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProxyHostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_TOKEN_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Token.Name == "" {
				return nil
			}
			return []string{ph.Spec.Token.Name}
		}); err != nil {
		return err
	}

	// Add the CustomCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_CUSTOM_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Ssl.CustomCertificate == nil || ph.Spec.Ssl.CustomCertificate.Name == "" {
				return nil
			}
			return []string{ph.Spec.Ssl.CustomCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the LetsEncryptCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_LETSENCRYPT_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Ssl.LetsEncryptCertificate == nil || ph.Spec.Ssl.LetsEncryptCertificate.Name == "" {
				return nil
			}
			return []string{ph.Spec.Ssl.LetsEncryptCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the Forward Service to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_FORWARD_SERVICE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if ph.Spec.Forward.Service == nil || ph.Spec.Forward.Service.Name == "" {
				return nil
			}
			return []string{ph.Spec.Forward.Service.Name}
		}); err != nil {
		return err
	}

	// Add the CustomLocation Forward Service to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.ProxyHost{},

		PH_CUSTOM_LOCATION_FORWARD_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			ph := rawObj.(*nginxpmoperatoriov1.ProxyHost)
			if len(ph.Spec.CustomLocations) == 0 {
				return nil
			}

			var fieldsList []string

			for _, location := range ph.Spec.CustomLocations {
				if location.Forward.Service == nil || location.Forward.Service.Name == "" {
					continue
				}

				// Append if not already present
				if !slices.Contains(fieldsList, location.Forward.Service.Name) {
					fieldsList = append(fieldsList, location.Forward.Service.Name)
				}
			}

			if len(fieldsList) == 0 {
				return nil
			}

			return fieldsList
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.ProxyHost{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Owns(&nginxpmoperatoriov1.CustomCertificate{}).
		Owns(&nginxpmoperatoriov1.LetsEncryptCertificate{}).
		Owns(&corev1.Service{}).
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(PH_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.CustomCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(PH_CUSTOM_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.LetsEncryptCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(PH_LETSENCRYPT_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(PH_FORWARD_SERVICE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(PH_CUSTOM_LOCATION_FORWARD_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *ProxyHostReconciler) findObjectsForObjects(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.ProxyHostList{}

		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, object.GetName()),
			Namespace:     object.GetNamespace(),
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
