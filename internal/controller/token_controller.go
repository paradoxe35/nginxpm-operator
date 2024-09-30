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
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
)

const (
	secretField = ".spec.secret.secretName"

	typeConditionReconciling = "Reconciling"

	typeConditionSecret = "Secret"

	typeConditionToken = "Token"

	typeConditionClientInstance = "ClientInstance"

	typeConditionCheckConnection = "CheckConnection"
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

	token := &nginxpmoperatoriov1.Token{}

	// Fetch the Token instance
	// The purpose is check if the Custom Resource for the Kind Token
	// is applied on the cluster if not we return nil to stop the reconciliation
	if err := r.Get(ctx, req.NamespacedName, token); err != nil {
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

	if !token.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Let's just set the status as Unknown when no status is available
	if len(token.Status.Conditions) == 0 {
		meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
			Type:    typeConditionReconciling,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})

		r.updateStatus(ctx, req, token)
	}

	// Let's create a new Nginx Proxy Manager client
	nginxpmClient, err := r.initNginxPMClient(ctx, req, token)
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("### Client Token Ready and Expires at: ", nginxpmClient.Expires, " ###")

	if token.Status.Token == nil || *token.Status.Token != nginxpmClient.Token {
		meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
			Type:    typeConditionToken,
			Status:  metav1.ConditionTrue,
			Reason:  "TokenCreated",
			Message: fmt.Sprintf("Token created successfully and expires at: %s", nginxpmClient.Expires.String()),
		})

		// Update the status of the token
		// Set the token and expiration time in the status
		token.Status.Token = &nginxpmClient.Token
		token.Status.Expires = &metav1.Time{Time: nginxpmClient.Expires}

		// Update the status of the token with the new token and expiration time
		r.updateStatus(ctx, req, token)
	}

	// Could be better to use the expiration time from the token status,
	// but this is a quick fix
	requeueAfter := nginxpmClient.Expires.UTC().Sub(metav1.Now().UTC())

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *TokenReconciler) initNginxPMClient(ctx context.Context, req reconcile.Request, token *nginxpmoperatoriov1.Token) (*Client, error) {
	log := logger.FromContext(ctx)

	// Get the secret resource associated with the token
	secret := &corev1.Secret{}
	secretName := token.Spec.Secret.SecretName
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: secretName}, secret); err != nil {
		// If the secret resource is not found, we will not be able to create the token
		log.Error(err, "Secret resource not found, please check the secret resource name")

		// Let's set the status of the token
		meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
			Type:    typeConditionSecret,
			Status:  metav1.ConditionFalse,
			Reason:  "SecretNotFound",
			Message: fmt.Sprintf("Secret resource not found, please check the secret resource name: %s", secretName),
		})

		r.updateStatus(ctx, req, token)

		return nil, err
	}

	// Let's check if the secret resource is valid
	identity, ok := secret.Data["identity"]
	if !ok {
		err := errors.New("failed to get secret from secret")
		log.Error(err, "failed to get secret from secret")
		return nil, err
	}

	// Let's check if the secret resource is valid
	secretDataValue, ok := secret.Data["secret"]
	if !ok {
		err := errors.New("failed to get secret from secret")
		log.Error(err, "failed to get secret from secret")
		return nil, err
	}

	// Let create a new Nginx Proxy Manager client
	var nginxpmClient *Client

	// Let's create a new HTTP client with a timeout
	httpClient := &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}

	// If the token is not empty, we will use it to create new client from
	expiredAt := token.Status.Expires
	hasValidToken := token.Status.Token != nil && expiredAt != nil && expiredAt.UTC().After(time.Now().UTC())

	// Let's set the status of the token
	meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
		Type:    typeConditionClientInstance,
		Status:  metav1.ConditionTrue,
		Reason:  "InitializingClient",
		Message: "Initializing Nginx Proxy Manager client object",
	})

	r.updateStatus(ctx, req, token)

	// If the token is valid, we will use it to create new client from
	if hasValidToken {
		log.Info("Using token from status")

		nginxpmClient = NewClientFromToken(httpClient, token)

		// Check if the connection is established
		if err := nginxpmClient.CheckConnection(); err != nil {
			log.Error(err, "Connect to the nginx-proxy-manager endpoint failed")

			// Let's set the status of the token
			meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
				Type:    typeConditionCheckConnection,
				Status:  metav1.ConditionFalse,
				Reason:  "CheckConnectionFailed",
				Message: fmt.Sprintf("Connect to the nginx-proxy-manager endpoint failed: %s", err.Error()),
			})

			r.updateStatus(ctx, req, token)

			return nil, err
		}
	}

	// If the token is empty, we will use the identity
	// and secret from secret to create new client from"
	if !hasValidToken {
		log.Info("Instantiating new nginxpm client and create token")

		// Let's create a new Nginx Proxy Manager client
		nginxpmClient = NewClient(httpClient, token.Spec.Endpoint)

		// Check if the connection is established
		if err := nginxpmClient.CheckConnection(); err != nil {
			log.Error(err, "Connect to the nginx-proxy-manager endpoint failed")

			// Let's set the status of the token
			meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
				Type:    typeConditionCheckConnection,
				Status:  metav1.ConditionFalse,
				Reason:  "CheckConnectionFailed",
				Message: fmt.Sprintf("Connect to the nginx-proxy-manager endpoint failed: %s", err.Error()),
			})

			r.updateStatus(ctx, req, token)

			return nil, err
		}

		// Let's create a new token from the identity and secret
		if err := CreateClientToken(nginxpmClient, string(identity), string(secretDataValue)); err != nil {
			log.Error(err, "Failed to create token from identity and secret")

			// Let's set the status of the token
			meta.SetStatusCondition(&token.Status.Conditions, metav1.Condition{
				Type:    typeConditionToken,
				Status:  metav1.ConditionFalse,
				Reason:  "FailedToCreateToken",
				Message: fmt.Sprintf("Failed to create token from identity and secret: %s", err.Error()),
			})

			r.updateStatus(ctx, req, token)

			return nil, err
		}
	}

	return nginxpmClient, nil
}

func (r *TokenReconciler) updateStatus(ctx context.Context, req ctrl.Request, token *nginxpmoperatoriov1.Token) error {
	log := logger.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Let's re-fetch the token Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster
		defer r.Get(ctx, req.NamespacedName, token)

		return r.Status().Update(ctx, token)
	})

	if err != nil {
		log.Error(err, "Failed to update Token status")
		return err
	}

	return nil
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
		Owns(&corev1.Secret{}).
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
