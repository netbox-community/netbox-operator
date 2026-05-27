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

package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestCalculateNextReconcile_PositiveSchedule(t *testing.T) {
	config.ResetForTesting()

	os.Setenv("NETBOX_HOST", "netbox_host")
	os.Setenv("AUTH_TOKEN", "auth-token")
	os.Setenv("RECONCILE_JITTER", "10s")
	os.Setenv("RECONCILE_SCHEDULE", "*/2 * * * *")

	maxJitter := config.GetOperatorConfig().ReconcileJitterDuration

	ctx := context.Background()
	result, err := CalculateNextReconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter > 2*time.Minute+maxJitter {
		t.Errorf("unexpected RequeueAfter: got %v, want between 0 min and 2 min 10 seconds", result.RequeueAfter)
	}
}

func TestCalculateNextReconcile_ZeroSchedule(t *testing.T) {
	config.ResetForTesting()

	os.Setenv("NETBOX_HOST", "netbox_host")
	os.Setenv("AUTH_TOKEN", "auth-token")
	os.Setenv("RECONCILE_JITTER", "1s")
	os.Setenv("RECONCILE_SCHEDULE", "")

	schedule := config.GetOperatorConfig().ReconcileSchedule
	if schedule != nil {
		t.Fatalf("expected empty schedule but got %v", schedule)
	}

	ctx := context.Background()
	result, err := CalculateNextReconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.True(t, result.IsZero())
}

func TestCalculateNextReconcile_ZeroJitter(t *testing.T) {
	config.ResetForTesting()

	os.Setenv("NETBOX_HOST", "netbox_host")
	os.Setenv("AUTH_TOKEN", "auth-token")
	// if empty jitter will be set to the default of 1 hour
	os.Setenv("RECONCILE_JITTER", "")
	os.Setenv("RECONCILE_SCHEDULE", "0 */2 * * *")

	next := config.GetOperatorConfig().ReconcileSchedule.Next(time.Now())

	ctx := context.Background()
	result, err := CalculateNextReconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter > 2*time.Hour+1*time.Hour {
		t.Errorf("unexpected RequeueAfter: got %v, want 2m (next: %v)", result.RequeueAfter, next)
	}
}

func TestGetJitterDuration_Range(t *testing.T) {
	config.ResetForTesting()

	os.Setenv("NETBOX_HOST", "netbox_host")
	os.Setenv("AUTH_TOKEN", "auth-token")
	os.Setenv("RECONCILE_JITTER", "5s")

	maxJitter := config.GetOperatorConfig().ReconcileJitterDuration

	for i := 0; i < 10; i++ {
		jitter := getJitterDuration()
		if jitter < 0 || jitter >= 5*time.Second {
			t.Errorf("jitter out of range: got %v, want between 0 s  and %v", jitter, maxJitter)
		}
	}
}
