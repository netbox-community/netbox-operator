/*
Copyright 2024 Swisscom (Schweiz) AG.

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
	"encoding/json"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func convertCIDRToLeaseLockName(cidr string) string {
	return strings.ReplaceAll(strings.ReplaceAll(cidr, "/", "-"), ":", "-")
}

func generateManagedCustomFieldsAnnotation(customFields map[string]string) (string, error) {
	if customFields == nil {
		customFields = make(map[string]string)
	}

	metadataJSON, err := json.Marshal(customFields)
	if err != nil {
		return "", fmt.Errorf("failed to marshal custom fields to JSON: %w", err)
	}

	return string(metadataJSON), nil
}

func removeFinalizer(ctx context.Context, c client.Client, o client.Object, finalizerName string) error {
	logger := log.FromContext(ctx)
	if controllerutil.ContainsFinalizer(o, finalizerName) {
		logger.V(4).Info("removing the finalizer")
		controllerutil.RemoveFinalizer(o, finalizerName)
		if err := c.Update(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

func addFinalizer(ctx context.Context, c client.Client, o client.Object, finalizerName string) error {
	logger := log.FromContext(ctx)
	if !controllerutil.ContainsFinalizer(o, finalizerName) {
		logger.V(4).Info("add the finalizer")
		controllerutil.AddFinalizer(o, finalizerName)
		if err := c.Update(ctx, o); err != nil {
			return err
		}
	}

	return nil
}
