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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"github.com/paradoxe35/nginxpm-operator/pkg/util"
)

const (
	customCertificateFinalizer = "customcertificate.nginxpm-operator.io/finalizers"

	CC_CERTIFICATE_FIELD = ".spec.certificate.secret.name"

	CC_TOKEN_FIELD = ".spec.token.name"
)

// CustomCertificateReconciler reconciles a CustomCertificate object
type CustomCertificateReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the CustomCertificate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *CustomCertificateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cc := &nginxpmoperatoriov1.CustomCertificate{}

	// Fetch the CustomCertificate instance
	// The purpose is check if the Custom Resource for the Kind CustomCertificate
	// is applied on the cluster if not we return nil to stop the reconciliation
	err := r.Get(ctx, req.NamespacedName, cc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("customCertificate resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get customCertificate")
		return ctrl.Result{}, err
	}

	isMarkedToBeDeleted := !cc.ObjectMeta.DeletionTimestamp.IsZero()

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	if !isMarkedToBeDeleted {
		if err := AddFinalizer(r, ctx, cc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create a new Nginx Proxy Manager client
	// If the client can't be created, we will remove the finalizer
	nginxpmClient, err := r.initNginxPMClient(ctx, cc)
	if err != nil {
		// If the can't initialize the client, we will remove the finalizer
		if err := RemoveFinalizer(r, ctx, cc); err != nil {
			return ctrl.Result{}, err
		}

		// Stop reconciliation if the resource is marked for deletion and the client can't be created
		if isMarkedToBeDeleted {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// If the resource is marked for deletion
	// Delete the CustomCertificate record in the Nginx Proxy Manager instance before deleting the resource
	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(cc, customCertificateFinalizer) {
			log.Info("Performing Finalizer Operations for CustomCertificate")

			// Delete the CustomCertificate record in the Nginx Proxy Manager instance
			if cc.Status.Id != nil {
				log.Info("Deleting CustomCertificate record in Nginx Proxy Manager instance")
				err := nginxpmClient.DeleteCertificate(int(*cc.Status.Id))

				if err != nil {
					log.Error(err, "Failed to delete CustomCertificate record in Nginx Proxy Manager instance")
				}
			}

			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, cc); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Create Certificate or update the existing one
	result, err := r.createCertificate(ctx, req, cc, nginxpmClient)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *CustomCertificateReconciler) createCertificate(ctx context.Context, req ctrl.Request, lec *nginxpmoperatoriov1.CustomCertificate, nginxpmClient *nginxpm.Client) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Code here .....

	return ctrl.Result{}, nil
}

// initNginxPMClient will create a new Nginx Proxy Manager client from the token resource
func (r *CustomCertificateReconciler) initNginxPMClient(ctx context.Context, cc *nginxpmoperatoriov1.CustomCertificate) (*nginxpm.Client, error) {
	log := log.FromContext(ctx)

	token := &nginxpmoperatoriov1.Token{}
	tokenName := types.NamespacedName{
		Namespace: cc.Spec.Token.Namespace,
		Name:      cc.Spec.Token.Name,
	}

	// Get the token resource
	if err := r.Get(ctx, tokenName, token); err != nil {
		log.Error(err, "Failed to get token resource")

		r.Recorder.Event(
			cc, "Warning", "GetToken",
			fmt.Sprintf("Failed to get token resource, ResourceName: %s, Namespace: %s", tokenName.Name, tokenName.Namespace),
		)
		return nil, err
	}

	// Create a new Nginx Proxy Manager client
	nginxpmClient := nginxpm.NewClientFromToken(util.NewHttpClient(), token)

	// Check if the connection is established
	if err := nginxpmClient.CheckTokenAccess(); err != nil {
		log.Error(err, "Token access check failed")

		r.Recorder.Event(
			cc, "Warning", "CheckTokenAccess",
			fmt.Sprintf("Failed to check token access, ResourceName: %s, Namespace: %s", tokenName.Name, tokenName.Namespace),
		)
		return nil, err
	}

	return nginxpmClient, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.CustomCertificate{}).
		Complete(r)
}
