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
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
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

type CustomCertificateKeys struct {
	Certificate    []byte
	CertificateKey []byte
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=customcertificates/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get
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
		if err := AddFinalizer(r, ctx, customCertificateFinalizer, cc); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create a new Nginx Proxy Manager client
	// If the client can't be created, we will remove the finalizer
	nginxpmClient, err := InitNginxPMClient(ctx, r, req, cc.Spec.Token)
	if err != nil {
		// Stop reconciliation if the resource is marked for deletion and the client can't be created
		if isMarkedToBeDeleted {
			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, customCertificateFinalizer, cc); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		} else {
			r.Recorder.Event(
				cc, "Warning", "InitNginxPMClient",
				fmt.Sprintf("Failed to init nginxpm client, ResourceName: %s, Namespace: %s", req.Name, req.Namespace),
			)
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
			if err := RemoveFinalizer(r, ctx, customCertificateFinalizer, cc); err != nil {
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

func (r *CustomCertificateReconciler) createCertificate(ctx context.Context, req ctrl.Request, cc *nginxpmoperatoriov1.CustomCertificate, nginxpmClient *nginxpm.Client) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var certificate *nginxpm.CustomCertificate
	var err error

	// Let's check if the CustomCertificate is already created
	if cc.Status.Id != nil {
		certificate, err = nginxpmClient.FindCustomCertificateByID(*cc.Status.Id)
		if err != nil {
			log.Error(err, "Failed to find CustomCertificate by ID")

			UpdateStatus(ctx, r.Client, cc, req.NamespacedName, func() {
				msg := "Failed to find CustomCertificate by ID"
				cc.Status.Status = &msg
			})

			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Let's create a new CustomCertificate from the CustomCertificate resource
	if cc.Status.Id == nil {
		log.Info("Creating CustomCertificate")

		// Retrieve the certificate and certificate key from the secret
		certificateKeys, err := r.getCertificateKeys(ctx, req, cc)
		if err != nil {
			UpdateStatus(ctx, r.Client, cc, req.NamespacedName, func() {
				msg := "Failed to retrieve certificate and certificate key"
				cc.Status.Status = &msg
			})

			return ctrl.Result{}, err
		}

		var niceName string = req.Name
		if cc.Spec.NiceName != nil && len(*cc.Spec.NiceName) > 0 {
			niceName = *cc.Spec.NiceName
		}

		r.Recorder.Event(
			cc, "Normal", "CreatingCustomCertificate",
			fmt.Sprintf("Creating CustomCertificate, Cert Name: %s, Namespace: %s", niceName, req.Namespace),
		)

		certificate, err = nginxpmClient.CreateCustomCertificate(
			nginxpm.CreateCustomCertificateRequest{
				NiceName:       niceName,
				Provider:       nginxpm.CUSTOM_PROVIDER,
				Certificate:    certificateKeys.Certificate,
				CertificateKey: certificateKeys.CertificateKey,
			},
		)

		if err != nil {
			log.Error(err, "Failed to create CustomCertificate")

			r.Recorder.Event(
				cc, "Warning", "CreateCustomCertificate",
				fmt.Sprintf("Failed to create CustomCertificate, Cert Name: %s, Namespace: %s", niceName, req.Namespace),
			)

			UpdateStatus(ctx, r.Client, cc, req.NamespacedName, func() {
				msg := "Failed to create CustomCertificate"
				cc.Status.Status = &msg
			})

			return ctrl.Result{RequeueAfter: time.Minute * 1}, nil
		}

		r.Recorder.Event(
			cc, "Normal", "CreatedCustomCertificate",
			fmt.Sprintf("Created CustomCertificate, Cert Name: %s, Namespace: %s", niceName, req.Namespace),
		)
	}

	return ctrl.Result{}, UpdateStatus(ctx, r.Client, cc, req.NamespacedName, func() {
		msg := "Certificate ready"

		cc.Status.Id = &certificate.ID
		cc.Status.ExpiresOn = &certificate.ExpiresOn
		cc.Status.Status = &msg
	})
}

func (r *CustomCertificateReconciler) getCertificateKeys(ctx context.Context, req ctrl.Request, cc *nginxpmoperatoriov1.CustomCertificate) (*CustomCertificateKeys, error) {
	log := log.FromContext(ctx)

	secret := &corev1.Secret{}
	// Retrieve the ProviderCredentials secret
	secretName := cc.Spec.Certificate.Secret.Name
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: secretName}, secret); err != nil {
		// If the secret resource is not found, we will not be able to create the token
		log.Error(err, "Secret resource not found, please check the secret resource name")

		r.Recorder.Event(
			cc, "Warning", "getCertificateKeys",
			fmt.Sprintf("Failed to get secret resource, ResourceName: %s, Namespace: %s", secretName, req.Namespace),
		)
		return nil, err
	}

	// Get the certificate from the secret data
	certificate, ok := secret.Data["certificate"]
	if !ok {
		err := errors.New("failed to get [certificate] field from secret")
		log.Error(err, "failed to get [certificate] field from secret")
		return nil, err
	}

	// Get the certificate key from the secret data
	certificateKey, ok := secret.Data["certificate_key"]
	if !ok {
		err := errors.New("failed to get [certificate_key] field from secret")
		log.Error(err, "failed to get [certificate_key] field from secret")
		return nil, err
	}

	return &CustomCertificateKeys{Certificate: certificate, CertificateKey: certificateKey}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&nginxpmoperatoriov1.CustomCertificate{},
		CC_TOKEN_FIELD,
		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			cc := rawObj.(*nginxpmoperatoriov1.CustomCertificate)

			if cc.Spec.Token == nil {
				// If token is not provided, use the default token name
				return []string{TOKEN_DEFAULT_NAME}
			}

			if cc.Spec.Token.Name == "" {
				return nil
			}

			return []string{cc.Spec.Token.Name}
		}); err != nil {
		return err
	}

	// Add certificate secret to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),

		&nginxpmoperatoriov1.CustomCertificate{},

		CC_CERTIFICATE_FIELD,

		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			cc := rawObj.(*nginxpmoperatoriov1.CustomCertificate)
			if cc.Spec.Certificate.Secret.Name == "" {
				return nil
			}

			return []string{cc.Spec.Certificate.Secret.Name}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.CustomCertificate{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(CC_CERTIFICATE_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(CC_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *CustomCertificateReconciler) findObjectsForObjects(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.CustomCertificateList{}

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
