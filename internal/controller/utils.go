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
	"errors"
	"fmt"
	"strings"

	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StatusError wraps an error that should update status conditions.
// Use NewStatusError to create errors that should be reflected in status.
type StatusError struct {
	err error
}

func (e *StatusError) Error() string {
	return e.err.Error()
}

func (e *StatusError) Unwrap() error {
	return e.err
}

// NewStatusError creates an error that will update the resource status condition.
// Use this for errors that should be visible in kubectl describe output.
func NewStatusError(format string, args ...interface{}) error {
	return &StatusError{err: fmt.Errorf(format, args...)}
}

// IsStatusError checks if an error should update status conditions.
func IsStatusError(err error) bool {
	var statusErr *StatusError
	return errors.As(err, &statusErr)
}

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

type EventStatusRecorder struct {
	client client.Client
	rec    record.EventRecorder
}

func NewEventStatusRecorder(client client.Client, rec record.EventRecorder) *EventStatusRecorder {
	return &EventStatusRecorder{
		client: client,
		rec:    rec,
	}
}

func (esr *EventStatusRecorder) Report(ctx context.Context, o ObjectWithConditions, condition metav1.Condition, eventType string, errExt error, additionalMessages ...string) error {
	logger := log.FromContext(ctx)

	if errExt != nil {
		condition.Message = condition.Message + ": " + errExt.Error()
		logger.Error(errExt, condition.Message)
	}

	condition.Message = strings.Join(append([]string{condition.Message}, additionalMessages...), ", ")
	condition.ObservedGeneration = o.GetGeneration()

	conditionChanged := apismeta.SetStatusCondition(o.Conditions(), condition)
	if conditionChanged {
		esr.rec.Event(o, eventType, condition.Reason, condition.Message)
		logger.Info("Condition "+condition.Type+" changed to "+string(condition.Status), "Reason", condition.Reason, "Message", condition.Message)

		err := esr.client.Status().Update(ctx, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (esr *EventStatusRecorder) Recorder() record.EventRecorder {
	return esr.rec
}
