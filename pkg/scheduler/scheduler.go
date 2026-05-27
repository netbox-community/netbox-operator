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

/*
Package scheduler contains scheduled reconciliation logic.
*/
package scheduler

import (
	"context"
	"math/rand"
	"time"

	"github.com/netbox-community/netbox-operator/pkg/config"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CalculateNextReconcile(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// do not reschedule if no schedule is defined
	if config.GetOperatorConfig().ReconcileSchedule == nil {
		logger.Info("Scheduled reconciliation disabled: no reconcile schedule configured")
		return ctrl.Result{}, nil
	}

	// Calculate duration till next reconciliation and add jitter
	jitter := getJitterDuration()
	now := time.Now()
	nextRunWithJitter := config.GetOperatorConfig().ReconcileSchedule.Next(now).Sub(now) + jitter
	if nextRunWithJitter < 0 {
		nextRunWithJitter = 0
	}

	logger.Info("Calculated next reconciliation delay",
		"nextRunWithJitter", nextRunWithJitter.String())

	if nextRunWithJitter > 0 {
		logger.Info("Scheduled next reconciliation",
			"after", nextRunWithJitter.String(),
			"jitter", jitter.String())
	}

	return ctrl.Result{RequeueAfter: nextRunWithJitter}, nil
}

func getJitterDuration() time.Duration {
	if config.GetOperatorConfig().ReconcileJitterDuration == 0 {
		return 0
	}

	return time.Duration(rand.Int63n(
		int64(config.GetOperatorConfig().ReconcileJitterDuration),
	))
}
