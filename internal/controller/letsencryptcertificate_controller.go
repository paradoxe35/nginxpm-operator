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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
)

const (
	letsEncryptCertificateFinalizer = "letsencryptcertificate.finalizers.nginxpm-operator.io"
)

// LetsEncryptCertificateReconciler reconciles a LetsEncryptCertificate object
type LetsEncryptCertificateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the LetsEncryptCertificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *LetsEncryptCertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	letsEncryptCertificate := &nginxpmoperatoriov1.LetsEncryptCertificate{}

	// Fetch the LetsEncryptCertificate instance
	// The purpose is check if the Custom Resource for the Kind LetsEncryptCertificate
	// is applied on the cluster if not we return nil to stop the reconciliation
	err := r.Get(ctx, req.NamespacedName, letsEncryptCertificate)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("letsEncryptCertificate resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get letsEncryptCertificate")
		return ctrl.Result{}, err
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(letsEncryptCertificate, letsEncryptCertificateFinalizer) {
		log.Info("Adding Finalizer for LetsEncryptCertificate")
		if ok := controllerutil.AddFinalizer(letsEncryptCertificate, letsEncryptCertificateFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, letsEncryptCertificate); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// ....

	return ctrl.Result{}, nil
}

func (r *LetsEncryptCertificateReconciler) updateStatus(letsEncryptCertificate *nginxpmoperatoriov1.LetsEncryptCertificate, ctx context.Context, req ctrl.Request, mutate func(status *nginxpmoperatoriov1.LetsEncryptCertificateStatus)) error {
	log := log.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(ctx, req.NamespacedName, letsEncryptCertificate)
		if err != nil {
			return err
		}

		mutate(&letsEncryptCertificate.Status)

		// Update the status of the LetsEncryptCertificate
		return r.Status().Update(ctx, letsEncryptCertificate)
	})

	if err != nil {
		log.Error(err, "Failed to update LetsEncryptCertificate status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LetsEncryptCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.LetsEncryptCertificate{}).
		Complete(r)
}
