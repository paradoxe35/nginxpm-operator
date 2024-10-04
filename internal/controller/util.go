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
