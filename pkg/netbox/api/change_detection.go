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

package api

import (
	"context"
	"time"

	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IsUpToDate(
	ctx context.Context,
	netboxLastUpdated time.Time,
	statusLastUpdated metav1.Time,
	conditions []metav1.Condition,
	generation int64,
) bool {
	logger := log.FromContext(ctx)
	if statusLastUpdated.IsZero() {
		return false
	}
	sameLastUpdated := statusLastUpdated.Time.Equal(netboxLastUpdated.Truncate(time.Second))
	if !sameLastUpdated {
		logger.Info("resource in NetBox not up to date, different lastUpdated in NetBox")
	}

	readyCondition := apismeta.FindStatusCondition(conditions, "Ready")
	readyForLatestGeneration := readyCondition != nil && readyCondition.Status == "True" && readyCondition.ObservedGeneration == generation
	if !readyForLatestGeneration {
		logger.Info("resource in NetBox not up to date, cr not ready for latest generation")
	}

	return sameLastUpdated && readyForLatestGeneration
}
