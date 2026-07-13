/*
Copyright 2026grep controller-runtime go.mod Swisscom (Schweiz) AG.

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

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PriorityHandler struct {
	handler.TypedEnqueueRequestForObject[client.Object]
}

// PriorityResource is implemented by any resource that carries a scheduling priority.
type PriorityResource interface {
	Priority() netboxv1.Priority
}

func priorityFromResource(r PriorityResource) *int {
	var p int
	switch r.Priority() {
	case netboxv1.PriorityCritical:
		p = 4
	case netboxv1.PriorityHigh:
		p = 3
	case netboxv1.PriorityMedium:
		p = 2
	case netboxv1.PriorityLow:
		p = 1
	default:
		p = 0
	}
	return &p
}

type priorityQueue[request comparable] struct {
	priorityqueue.PriorityQueue[request]
	priority *int
}

func (w priorityQueue[request]) Add(item request) {
	w.PriorityQueue.AddWithOpts(
		priorityqueue.AddOpts{
			Priority: w.priority,
		},
		item,
	)
}

func WithPriority(
	h handler.TypedEventHandler[
		client.Object,
		reconcile.Request,
	],
) handler.TypedEventHandler[
	client.Object,
	reconcile.Request,
] {
	return handler.TypedFuncs[
		client.Object,
		reconcile.Request,
	]{
		CreateFunc: func(
			ctx context.Context,
			e event.TypedCreateEvent[client.Object],
			q workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			pq, ok := q.(priorityqueue.PriorityQueue[reconcile.Request])
			if !ok || e.Object == nil {
				h.Create(ctx, e, q)
				return
			}

			wrapped := priorityQueue[reconcile.Request]{
				PriorityQueue: pq,
				priority:      priorityFromResource(e.Object.(PriorityResource)),
			}

			h.Create(ctx, e, wrapped)
		},

		UpdateFunc: func(
			ctx context.Context,
			e event.TypedUpdateEvent[client.Object],
			q workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			pq, ok := q.(priorityqueue.PriorityQueue[reconcile.Request])
			if !ok || e.ObjectNew == nil {
				h.Update(ctx, e, q)
				return
			}

			wrapped := priorityQueue[reconcile.Request]{
				PriorityQueue: pq,
				priority:      priorityFromResource(e.ObjectNew.(PriorityResource)),
			}

			h.Update(ctx, e, wrapped)
		},

		DeleteFunc: h.Delete,

		GenericFunc: h.Generic,
	}
}
