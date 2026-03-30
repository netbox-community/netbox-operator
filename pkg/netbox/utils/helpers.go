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

package utils

import (
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SkipsUpdate(netboxLastUpdatedIsSet bool,
	statusLastUpdated *metav1.Time,
	conditions []metav1.Condition,
	generation int64,
	lastUpdatedEqual func(statusLastUpdated *metav1.Time) bool,
) bool {
	sameLastUpdated := netboxLastUpdatedIsSet == (statusLastUpdated != nil) &&
		(!netboxLastUpdatedIsSet || lastUpdatedEqual(statusLastUpdated))

	return sameLastUpdated && sameReadyGeneration(conditions, generation)
}

func sameReadyGeneration(conditions []metav1.Condition, generation int64) bool {
	readyCondition := apismeta.FindStatusCondition(conditions, "Ready")
	return readyCondition != nil && readyCondition.Status == "True" && readyCondition.ObservedGeneration == generation
}
