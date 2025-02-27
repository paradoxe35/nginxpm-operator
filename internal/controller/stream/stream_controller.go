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

package stream

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/internal/controller"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
)

const (
	streamFinalizer = "stream.nginxpm-operator.io/finalizers"

	ST_TOKEN_FIELD = ".spec.token.name"

	ST_CUSTOM_CERTIFICATE_FIELD = ".spec.ssl.customCertificate.name"

	ST_LETSENCRYPT_CERTIFICATE_FIELD = ".spec.ssl.letsEncryptCertificate.name"

	ST_FORWARD_SERVICE_FIELD = ".spec.forward.service.name"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=streams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=streams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=streams/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/status,verbs=get
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Stream object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	st := &nginxpmoperatoriov1.Stream{}

	err := r.Get(ctx, req.NamespacedName, st)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("stream resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get stream")
		return ctrl.Result{}, err
	}

	isMarkedToBeDeleted := !st.ObjectMeta.DeletionTimestamp.IsZero()

	if !isMarkedToBeDeleted {
		if err := controller.AddFinalizer(r, ctx, streamFinalizer, st); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Let's just set the status as Unknown when no status is available
	if len(st.Status.Conditions) == 0 {
		controller.UpdateStatus(ctx, r.Client, st, req.NamespacedName, func() {
			meta.SetStatusCondition(&st.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionUnknown,
				Type:               controller.ConditionTypeReconciling,
				Reason:             "Reconciling",
				Message:            "Starting reconciliation",
				LastTransitionTime: metav1.Now(),
			})
		})
	}

	// Create a new Nginx Proxy Manager client
	nginxpmClient, err := controller.InitNginxPMClient(ctx, r, req, st.Spec.Token)
	if err != nil {
		if isMarkedToBeDeleted {
			if err := controller.RemoveFinalizer(r, ctx, streamFinalizer, st); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		r.Recorder.Event(
			st, "Warning", "InitNginxPMClient",
			fmt.Sprintf("Failed to init nginxpm client: ResourceName: %s, Namespace: %s, err: %s",
				req.Name, req.Namespace, err.Error()),
		)

		// Set the status as False when the client can't be created
		controller.UpdateStatus(ctx, r.Client, st, req.NamespacedName, func() {
			meta.SetStatusCondition(&st.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionFalse,
				Type:               controller.ConditionTypeError,
				Reason:             "InitNginxPMClient",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
		})

		return ctrl.Result{}, err
	}

	// Create or update proxy host
	err = r.createOrUpdateStream(ctx, req, st, nginxpmClient)
	if err != nil {
		// Set the status as False when the client can't be created
		controller.UpdateStatus(ctx, r.Client, st, req.NamespacedName, func() {
			meta.SetStatusCondition(&st.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionFalse,
				Type:               controller.ConditionTypeError,
				Reason:             "createOrUpdateStream",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
		})

		return ctrl.Result{}, err
	}

	// Set the status as True when the client can be created
	controller.UpdateStatus(ctx, r.Client, st, req.NamespacedName, func() {
		meta.SetStatusCondition(&st.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionTrue,
			Type:               controller.ConditionTypeReady,
			Reason:             "createOrUpdateStream",
			Message:            fmt.Sprintf("Proxy host created or updated, ResourceName: %s", req.Name),
			LastTransitionTime: metav1.Now(),
		})
	})

	return ctrl.Result{}, nil
}

func (r *StreamReconciler) createOrUpdateStream(ctx context.Context, req ctrl.Request, st *nginxpmoperatoriov1.Stream, nginxpmClient *nginxpm.Client) error {
	// log := log.FromContext(ctx)

	// var stream *nginxpm.ProxyHost
	// var err error

	return nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *StreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.Stream{},

		ST_TOKEN_FIELD,

		func(rawObj client.Object) []string {
			st := rawObj.(*nginxpmoperatoriov1.Stream)

			if st.Spec.Token == nil {
				// If token is not provided, use the default token name
				return []string{controller.TOKEN_DEFAULT_NAME}
			}

			if st.Spec.Token.Name == "" {
				return nil
			}

			return []string{st.Spec.Token.Name}
		}); err != nil {
		return err
	}

	// Add the CustomCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.Stream{},

		ST_CUSTOM_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			st := rawObj.(*nginxpmoperatoriov1.Stream)
			if st.Spec.Ssl == nil || st.Spec.Ssl.CustomCertificate == nil || st.Spec.Ssl.CustomCertificate.Name == "" {
				return nil
			}
			return []string{st.Spec.Ssl.CustomCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the LetsEncryptCertificate to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.Stream{},

		ST_LETSENCRYPT_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			st := rawObj.(*nginxpmoperatoriov1.Stream)
			if st.Spec.Ssl == nil || st.Spec.Ssl.LetsEncryptCertificate == nil || st.Spec.Ssl.LetsEncryptCertificate.Name == "" {
				return nil
			}
			return []string{st.Spec.Ssl.LetsEncryptCertificate.Name}
		}); err != nil {
		return err
	}

	// Add the Forward Service to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.Stream{},

		ST_FORWARD_SERVICE_FIELD,

		func(rawObj client.Object) []string {
			st := rawObj.(*nginxpmoperatoriov1.Stream)
			if st.Spec.Forward.Service == nil || st.Spec.Forward.Service.Name == "" {
				return nil
			}
			return []string{st.Spec.Forward.Service.Name}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.Stream{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Owns(&nginxpmoperatoriov1.CustomCertificate{}).
		Owns(&nginxpmoperatoriov1.LetsEncryptCertificate{}).
		Owns(&corev1.Service{}).
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(ST_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.CustomCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(ST_CUSTOM_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.LetsEncryptCertificate{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(ST_LETSENCRYPT_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForMap(ST_FORWARD_SERVICE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.findStreamsForPod),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("stream").
		Complete(r)
}

func (r *StreamReconciler) findObjectsForMap(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.StreamList{}

		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, object.GetName()),
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

func (r *StreamReconciler) findStreamsForPod(ctx context.Context, obj client.Object) []reconcile.Request {
	pod := obj.(*corev1.Pod)
	log := log.FromContext(ctx)

	// Get all Stream resources
	streams := &nginxpmoperatoriov1.StreamList{}
	if err := r.List(ctx, streams); err != nil {
		log.Error(err, "Unable to list Stream resources")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request

	// For each Stream, check if the pod is associated with the referenced service
	for _, st := range streams.Items {
		// Skip if no service is specified
		if st.Spec.Forward.Service == nil || st.Spec.Forward.Service.Name == "" {
			continue
		}

		// Get the service
		service := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      st.Spec.Forward.Service.Name,
			Namespace: st.GetNamespace(),
		}, service); err != nil {
			// Service not found, skip
			continue
		}

		// Check if the pod's labels match the service's selector
		if controller.PodMatchesServiceSelector(pod, service) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      st.GetName(),
					Namespace: st.GetNamespace(),
				},
			})
			// Once we've added this Stream, we don't need to check other services
			// referenced in CustomLocations since we'll reconcile the entire Stream anyway
			continue
		}
	}

	return requests
}
