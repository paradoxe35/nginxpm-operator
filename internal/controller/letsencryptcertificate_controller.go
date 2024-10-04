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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
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
	"github.com/paradoxe35/nginxpm-operator/pkg/util"
	"k8s.io/apimachinery/pkg/types"
)

const (
	LEC_DNS_CHALLENGE_CRED_SECRET_FIELD = ".spec.dnsChallenge.providerCredentials.secret.name"

	LEC_TOKEN_FIELD = ".spec.token.name"

	letsEncryptCertificateFinalizer = "letsencryptcertificate.nginxpm-operator.io/finalizers"
)

// LetsEncryptCertificateReconciler reconciles a LetsEncryptCertificate object
type LetsEncryptCertificateReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=letsencryptcertificates/finalizers,verbs=update
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	lec := &nginxpmoperatoriov1.LetsEncryptCertificate{}

	// Fetch the LetsEncryptCertificate instance
	// The purpose is check if the Custom Resource for the Kind LetsEncryptCertificate
	// is applied on the cluster if not we return nil to stop the reconciliation
	err := r.Get(ctx, req.NamespacedName, lec)
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

	isMarkedToBeDeleted := !lec.ObjectMeta.DeletionTimestamp.IsZero()

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	if !isMarkedToBeDeleted {
		if err := AddFinalizer(r, ctx, letsEncryptCertificateFinalizer, lec); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create a new Nginx Proxy Manager client
	// If the client can't be created, we will remove the finalizer
	nginxpmClient, err := r.initNginxPMClient(ctx, lec)
	if err != nil {
		// Stop reconciliation if the resource is marked for deletion and the client can't be created
		if isMarkedToBeDeleted {
			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, letsEncryptCertificateFinalizer, lec); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// If the resource is marked for deletion
	// Delete the LetsEncryptCertificate record in the Nginx Proxy Manager instance before deleting the resource
	if isMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(lec, letsEncryptCertificateFinalizer) {
			log.Info("Performing Finalizer Operations for LetsEncryptCertificate")

			// Delete the LetsEncryptCertificate record in the Nginx Proxy Manager instance
			// If the LetsEncryptCertificate is bound, we will not delete the record
			if lec.Status.Id != nil && !lec.Status.Bound {
				log.Info("Deleting LetsEncryptCertificate record in Nginx Proxy Manager instance")
				err := nginxpmClient.DeleteCertificate(int(*lec.Status.Id))

				if err != nil {
					log.Error(err, "Failed to delete LetsEncryptCertificate record in Nginx Proxy Manager instance")
				}
			}

			// Remove the finalizer
			if err := RemoveFinalizer(r, ctx, letsEncryptCertificateFinalizer, lec); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Create Certificate or update the existing one
	result, err := r.createCertificate(ctx, req, lec, nginxpmClient)
	if err != nil {
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *LetsEncryptCertificateReconciler) createCertificate(ctx context.Context, req ctrl.Request, lec *nginxpmoperatoriov1.LetsEncryptCertificate, nginxpmClient *nginxpm.Client) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var certificate *nginxpm.LetsEncryptCertificate
	var err error

	// Convert domain names to []string
	domains := make([]string, len(lec.Spec.DomainNames))
	for i, domain := range lec.Spec.DomainNames {
		domains[i] = string(domain)
	}

	// Let's check if the LetsEncryptCertificate is already created
	if lec.Status.Id != nil {
		certificate, err = nginxpmClient.FindLetEncryptCertificateByID(*lec.Status.Id)
		if err != nil {
			log.Error(err, "Failed to find LetsEncryptCertificate by ID")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Let's create a new LetsEncryptCertificate from the LetsEncryptCertificate resource
	if lec.Status.Id == nil {
		log.Info("Creating LetsEncryptCertificate")

		hasDnsChallengeEnabled := lec.Spec.DnsChallenge != nil

		var credentials string
		var dnsChallengeProvider string

		if hasDnsChallengeEnabled {
			dnsChallengeProvider = lec.Spec.DnsChallenge.Provider

			// Retrieve the ProviderCredentials secret
			credentials, err = r.getDnsChallengeProviderCredentials(ctx, req, lec)
			if err != nil {
				return ctrl.Result{}, err
			}

		}

		r.Recorder.Event(
			lec, "Normal", "CreatingLetsEncryptCertificate",
			fmt.Sprintf("Creating LetsEncryptCertificate for domains %s, ResourceName: %s, Namespace: %s", strings.Join(domains, ","), req.Name, req.Namespace),
		)

		certificate, err = nginxpmClient.CreateLetEncryptCertificate(
			nginxpm.CreateLetEncryptCertificateRequest{
				DomainNames: domains,
				Provider:    nginxpm.LETSENCRYPT_PROVIDER,
				Meta: nginxpm.CreateLetEncryptCertificateRequestMeta{
					DNSChallenge:           hasDnsChallengeEnabled,
					DNSProvider:            dnsChallengeProvider,
					DNSProviderCredentials: credentials,
					LetsEncryptAgree:       true,
					LetsEncryptEmail:       lec.Spec.LetsEncryptEmail,
				},
			},
		)

		if err != nil {
			log.Error(err, "Failed to create LetsEncryptCertificate")

			r.Recorder.Event(
				lec, "Warning", "CreateLetsEncryptCertificate",
				fmt.Sprintf("Failed to create LetsEncryptCertificate for domains %s, ResourceName: %s, Namespace: %s", strings.Join(domains, ","), req.Name, req.Namespace),
			)

			return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
		}

		r.Recorder.Event(
			lec, "Normal", "CreatedLetsEncryptCertificate",
			fmt.Sprintf("Created LetsEncryptCertificate for domains %s, ResourceName: %s, Namespace: %s", strings.Join(domains, ","), req.Name, req.Namespace),
		)

		// Update bound status only if the LetsEncryptCertificate is created
		return ctrl.Result{}, r.updateStatus(lec, ctx, req, func(status *nginxpmoperatoriov1.LetsEncryptCertificateStatus) {
			status.Bound = certificate.Bound
			status.Id = &certificate.ID
			status.DomainNames = certificate.DomainNames
			status.ExpiresOn = &certificate.ExpiresOn
		})
	}

	// Update the LetsEncryptCertificate status
	return ctrl.Result{}, r.updateStatus(lec, ctx, req, func(status *nginxpmoperatoriov1.LetsEncryptCertificateStatus) {
		status.Id = &certificate.ID
		status.DomainNames = certificate.DomainNames
		status.ExpiresOn = &certificate.ExpiresOn
	})
}

func (r *LetsEncryptCertificateReconciler) getDnsChallengeProviderCredentials(ctx context.Context, req ctrl.Request, lec *nginxpmoperatoriov1.LetsEncryptCertificate) (string, error) {
	log := log.FromContext(ctx)

	var credentialsValue string

	if lec.Spec.DnsChallenge == nil {
		return credentialsValue, nil
	}

	secret := &corev1.Secret{}
	// Retrieve the ProviderCredentials secret
	secretName := lec.Spec.DnsChallenge.ProviderCredentials.Secret.Name
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: secretName}, secret); err != nil {
		// If the secret resource is not found, we will not be able to create the token
		log.Error(err, "Secret resource not found, please check the secret resource name")

		r.Recorder.Event(
			lec, "Warning", "GetDnsChallengeProviderCredentials",
			fmt.Sprintf("Failed to get secret resource, ResourceName: %s, Namespace: %s", secretName, req.Namespace),
		)
		return credentialsValue, err
	}

	// Let's check if the secret resource is valid
	credentials, ok := secret.Data["credentials"]
	if !ok {
		err := errors.New("failed to get secret from secret")
		log.Error(err, "failed to get secret from secret")
		return credentialsValue, err
	}

	credentialsValue = string(credentials)

	return credentialsValue, nil
}

// initNginxPMClient will create a new Nginx Proxy Manager client from the token resource
func (r *LetsEncryptCertificateReconciler) initNginxPMClient(ctx context.Context, lec *nginxpmoperatoriov1.LetsEncryptCertificate) (*nginxpm.Client, error) {
	log := log.FromContext(ctx)

	token := &nginxpmoperatoriov1.Token{}
	tokenName := types.NamespacedName{
		Namespace: lec.Spec.Token.Namespace,
		Name:      lec.Spec.Token.Name,
	}

	// Get the token resource
	if err := r.Get(ctx, tokenName, token); err != nil {
		log.Error(err, "Failed to get token resource")

		r.Recorder.Event(
			lec, "Warning", "GetToken",
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
			lec, "Warning", "CheckTokenAccess",
			fmt.Sprintf("Failed to check token access, ResourceName: %s, Namespace: %s", tokenName.Name, tokenName.Namespace),
		)
		return nil, err
	}

	log.Info("NginxPM client initialized successfully")

	return nginxpmClient, nil
}

func (r *LetsEncryptCertificateReconciler) updateStatus(lec *nginxpmoperatoriov1.LetsEncryptCertificate, ctx context.Context, req ctrl.Request, mutate func(status *nginxpmoperatoriov1.LetsEncryptCertificateStatus)) error {
	log := log.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(ctx, req.NamespacedName, lec)
		if err != nil {
			return err
		}

		mutate(&lec.Status)

		// Update the object status
		return r.Status().Update(ctx, lec)
	})

	if err != nil {
		log.Error(err, "Failed to update LetsEncryptCertificate status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LetsEncryptCertificateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add the DNS Challenge Provider Credentials Secret to the indexer
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &nginxpmoperatoriov1.LetsEncryptCertificate{}, LEC_DNS_CHALLENGE_CRED_SECRET_FIELD,
		func(rawObj client.Object) []string {
			// Extract the Secret name from the Token Spec, if one is provided
			lec := rawObj.(*nginxpmoperatoriov1.LetsEncryptCertificate)
			if lec.Spec.DnsChallenge == nil || lec.Spec.DnsChallenge.ProviderCredentials.Secret.Name == "" {
				return nil
			}

			return []string{lec.Spec.DnsChallenge.ProviderCredentials.Secret.Name}
		}); err != nil {
		return err
	}

	// Add the Token to the indexer
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nginxpmoperatoriov1.LetsEncryptCertificate{}, LEC_TOKEN_FIELD, func(rawObj client.Object) []string {
		// Extract the Secret name from the Token Spec, if one is provided
		lec := rawObj.(*nginxpmoperatoriov1.LetsEncryptCertificate)
		if lec.Spec.Token.Name == "" {
			return nil
		}
		return []string{lec.Spec.Token.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.LetsEncryptCertificate{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(LEC_DNS_CHALLENGE_CRED_SECRET_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&nginxpmoperatoriov1.Token{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForObjects(LEC_TOKEN_FIELD)),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *LetsEncryptCertificateReconciler) findObjectsForObjects(field string) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		attachedObjects := &nginxpmoperatoriov1.LetsEncryptCertificateList{}

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
