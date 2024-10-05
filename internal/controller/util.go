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

	nginxpmoperatoriov1 "github.com/paradoxe35/nginxpm-operator/api/v1"
	"github.com/paradoxe35/nginxpm-operator/pkg/nginxpm"
	"github.com/paradoxe35/nginxpm-operator/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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

func InitNginxPMClient(ctx context.Context, r client.Reader, tokenName string, tokenNamespace string) (*nginxpm.Client, error) {
	log := log.FromContext(ctx)

	token := &nginxpmoperatoriov1.Token{}
	tokenNamespaced := types.NamespacedName{
		Namespace: tokenNamespace,
		Name:      tokenName,
	}

	// Get the token resource
	if err := r.Get(ctx, tokenNamespaced, token); err != nil {
		log.Error(err, "Failed to get token resource")
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
