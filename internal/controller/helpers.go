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
	"reflect"
	"strings"

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"github.com/paradoxe35/nginxpm-operator/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// ConditionTypeReady indicates if the Resource is ready
	ConditionTypeReady = "Ready"

	// ConditionTypeReady indicates if the Resource is Reconciling (Unknown)
	ConditionTypeReconciling = "Reconciling"

	// ConditionTypeError indicates if there's an error with the Resource
	ConditionTypeError = "Error"
)

const (
	TOKEN_DEFAULT_NAME = "token-nginxpm"

	// Default namespaces to lookup for the token
	// when no specific namespace is provided
	TOKEN_SYSTEM_NAMESPACE  = "nginxpm-operator-system"
	TOKEN_DEFAULT_NAMESPACE = "default"
)

func InitNginxPMClient(ctx context.Context, r client.Reader, req reconcile.Request, tokenName *nginxpmoperatoriov1.TokenName) (*nginxpm.Client, error) {
	log := log.FromContext(ctx)

	// Set the token names
	names := []string{TOKEN_DEFAULT_NAME}
	if tokenName != nil && len(tokenName.Name) > 0 {
		// Prepend token name
		names = append([]string{tokenName.Name}, names...)
	}

	// Set the token namespaces
	namespaces := []string{req.Namespace, TOKEN_SYSTEM_NAMESPACE, TOKEN_DEFAULT_NAMESPACE}
	if tokenName != nil && tokenName.Namespace != nil && len(*tokenName.Namespace) > 0 {
		// Prepend token namespace
		namespaces = append([]string{*tokenName.Namespace}, namespaces...)
	}

	token := &nginxpmoperatoriov1.Token{}

	for _, namespace := range namespaces {
		found := false
		for _, name := range names {
			tokenNamespaced := types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}

			// Get the token resource
			err := r.Get(ctx, tokenNamespaced, token)
			if err == nil {
				log.Info("Token resource found", "Namespace", namespace, "Name", name)
				found = true
				break
			}
		}

		// token found on this iteration
		if found {
			break
		}
	}

	// If token still empty, means it was not found
	if token.Name == "" || token.Status.Token == nil {
		err := errors.New("token resource not found")
		log.Error(err, "Token resource not found")
		return nil, err
	}

	// Create a new Nginx Proxy Manager client
	nginxpmClient := nginxpm.NewClientFromToken(util.NewHttpClient(), token)

	// Check if the connection is established
	if err := nginxpmClient.CheckTokenAccess(); err != nil {
		log.Error(err, "Token access check failed")
		return nil, err
	}

	log.Info("NginxPM client initialized successfully")

	return nginxpmClient, nil
}

// RemoveFinalizer will remove the finalizer from the object
func RemoveFinalizer(r client.Writer, ctx context.Context, finalizer string, object client.Object) error {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(object, finalizer) {
		controllerutil.RemoveFinalizer(object, finalizer)

		if err := r.Update(ctx, object); err != nil {
			log.Error(err, "Failed to update custom resource to remove finalizer")
			return err
		}
	}

	return nil
}

// AddFinalizer will add a finalizer to the object
func AddFinalizer(r client.Writer, ctx context.Context, finalizer string, object client.Object) error {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(object, finalizer) {
		controllerutil.AddFinalizer(object, finalizer)

		if err := r.Update(ctx, object); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return err
		}
	}

	return nil
}

func UpdateStatus(ctx context.Context, r client.Client, object client.Object, namespacedName types.NamespacedName, mutate func()) error {
	log := log.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Get(ctx, namespacedName, object)
		if err != nil {
			return err
		}

		mutate()

		// Update the object status
		return r.Status().Update(ctx, object)
	})

	if err != nil {
		log.Error(err, "Failed to update resource status")
		return err
	}

	return nil
}

func JsonFieldExists(obj interface{}, field string) bool {
	// Check if obj is nil
	if obj == nil {
		return false
	}

	val := reflect.ValueOf(obj)

	// If it's a pointer, get the underlying element
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return false
		}
		val = val.Elem()
	}

	// Make sure we're working with a struct
	if val.Kind() != reflect.Struct {
		return false
	}

	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		jsonTag := structField.Tag.Get("json")

		// Parse the JSON tag to get just the name part (before any comma)
		jsonName := strings.Split(jsonTag, ",")[0]

		// Handle the case where the json tag is "-" (meaning skip this field)
		if jsonName == "-" {
			continue
		}

		// If the json tag is empty, use the field name
		if jsonName == "" {
			jsonName = structField.Name
		}

		// Check if this field matches the requested field name
		if jsonName == field {
			fieldValue := val.Field(i)

			if fieldValue.Kind() == reflect.Ptr || fieldValue.Kind() == reflect.Slice {
				return !fieldValue.IsNil()
			}

			return true
		}
	}

	return false
}
