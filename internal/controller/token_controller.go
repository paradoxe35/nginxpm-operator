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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
)

const (
	secretField = ".spec.secret.secretName"

	// typeAvailableToken represents the status of the Deployment reconciliation
	typeAvailableToken = "Available"
	// typeDegradedToken represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	typeDegradedToken = "Degraded"
	// tokenFinalizer is the name of the finalizer used to delete the custom resource
	tokenFinalizer = "nginxpm-operator.io/finalizer"
)

// TokenReconciler reconciles a Token object
type TokenReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nginxpm-operator.io,resources=tokens/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *TokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	// Fetch the Token instance
	// The purpose is check if the Custom Resource for the Kind Token
	// is applied on the cluster if not we return nil to stop the reconciliation
	token := &nginxpmoperatoriov1.Token{}
	err := r.Get(ctx, req.NamespacedName, token)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("token resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get token")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if len(token.Status.Conditions) == 0 {
		meta.SetStatusCondition(
			&token.Status.Conditions,
			metav1.Condition{
				Type:   typeAvailableToken,
				Status: metav1.ConditionUnknown,
				Reason: "Reconciling", Message: "Starting reconciliation",
			},
		)
		if err = r.Status().Update(ctx, token); err != nil {
			log.Error(err, "Failed to update Token status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the token Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, token); err != nil {
			log.Error(err, "Failed to re-fetch token")
			return ctrl.Result{}, err
		}
	}

	// Let's create a new Nginx Proxy Manager client
	nginxpmClient, err := r.initNginxPMClient(ctx, req.Namespace, token, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer(token, tokenFinalizer) {
		log.Info("Adding Finalizer for Token")
		if ok := controllerutil.AddFinalizer(token, tokenFinalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err = r.Update(ctx, token); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the Token instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isTokenMarkedToBeDeleted := token.GetDeletionTimestamp() != nil
	if isTokenMarkedToBeDeleted {

		if controllerutil.ContainsFinalizer(token, tokenFinalizer) {
			log.Info("Performing Finalizer Operations for Token before delete CR")

			// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
			meta.SetStatusCondition(
				&token.Status.Conditions,
				metav1.Condition{
					Type:    typeDegradedToken,
					Status:  metav1.ConditionUnknown,
					Reason:  "Finalizing",
					Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", token.Name),
				},
			)

			if err := r.Status().Update(ctx, token); err != nil {
				log.Error(err, "Failed to update Token status")
				return ctrl.Result{}, err
			}

			// Implement deletion logic here.....
		}
	}

	// if token.Spec.Expires.Before(&metav1.Now()) {
	// 	log.Info("Token is expired")
	// 	return ctrl.Result{}, nil
	// }

	return ctrl.Result{}, nil
}

func (r *TokenReconciler) initNginxPMClient(ctx context.Context, Namespace string, token *nginxpmoperatoriov1.Token, log logr.Logger) (*nginxpm.Client, error) {
	// Get the secret resource associated with the token
	secret := &corev1.Secret{}
	secretName := token.Spec.Secret.SecretName
	if err := r.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: secretName}, secret); err != nil {
		// If the secret resource is not found, we will not be able to create the token
		log.Error(err, "Secret resource not found, please check the secret resource name")
		return nil, err
	}

	// Let's check if the secret resource is valid
	identity, ok := secret.Data["identity"]
	if !ok {
		err := errors.New("Failed to get secret from secret")
		log.Error(err, "Failed to get secret from secret")
		return nil, err
	}

	// Let's check if the secret resource is valid
	secretDataValue, ok := secret.Data["secret"]
	if !ok {
		err := errors.New("Failed to get secret from secret")
		log.Error(err, "Failed to get secret from secret")
		return nil, err
	}

	// Let's check if the secret resource is valid
	// Let create a new Nginx Proxy Manager client
	// And check if the endpoint is valid and the connection is established
	// If token from status is not empty, we will use it to create new client from
	var nginxpmClient *nginxpm.Client

	// Let's create a new HTTP client with a timeout
	httpClient := &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}

	// If the token is not empty, we will use it to create new client from
	expires := token.Status.Expires
	hasValidToken := token.Status.Token != nil && expires != nil && expires.UTC().Before(time.Now().UTC())

	// If the token is valid, we will use it to create new client from
	if hasValidToken {
		log.Info("Using token from status")
		nginxpmClient = nginxpm.NewClientFromToken(httpClient, token.Spec.Endpoint, *token.Status.Token)
	}

	// If the token is empty, we will use the identity and secret from secret to create new client from"
	if !hasValidToken {
		log.Info("Using token from secret")

		// Let's decode the identity and secret from secret
		decodedIdentity, err := base64.StdEncoding.DecodeString(string(identity))
		if err != nil {
			log.Error(err, "Failed to decode identity from secret")
			return nil, err
		}

		decodedSecret, err := base64.StdEncoding.DecodeString(string(secretDataValue))
		if err != nil {
			log.Error(err, "Failed to decode secret from secret")
			return nil, err
		}

		// Let's create a new Nginx Proxy Manager client
		nginxpmClient = nginxpm.NewClient(httpClient, token.Spec.Endpoint)

		// Let's create a new token from the identity and secret
		if err := nginxpm.CreateClientToken(nginxpmClient, string(decodedIdentity), string(decodedSecret)); err != nil {
			log.Error(err, "Failed to create token from identity and secret")
			return nil, err
		}

		if err := nginxpmClient.CheckConnection(); err != nil {
			log.Error(err, "Connect to the nginx-proxy-manager endpoint failed")
			return nil, err
		}
	}

	return nginxpmClient, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TokenReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nginxpmoperatoriov1.Token{}, secretField, func(rawObj client.Object) []string {
		// Extract the ConfigMap name from the ConfigDeployment Spec, if one is provided
		configDeployment := rawObj.(*nginxpmoperatoriov1.Token)
		if configDeployment.Spec.Secret.SecretName == "" {
			return nil
		}
		return []string{configDeployment.Spec.Secret.SecretName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nginxpmoperatoriov1.Token{}).
		Owns(&nginxpmoperatoriov1.Token{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *TokenReconciler) findObjectsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	attachedSecrets := &nginxpmoperatoriov1.TokenList{}

	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(secretField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}

	err := r.List(ctx, attachedSecrets, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedSecrets.Items))
	for i, item := range attachedSecrets.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}

	return requests
}
