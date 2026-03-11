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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	serviceLoadBalancerManagedByLabel        = "netbox.dev/managed-by"
	serviceLoadBalancerManagedByValue        = "service-loadbalancer"
	serviceLoadBalancerServiceUIDLabel       = "netbox.dev/service-uid"
	serviceLoadBalancerServiceNameLabel      = "netbox.dev/service-name"
	serviceLoadBalancerServiceNamespaceLabel = "netbox.dev/service-namespace"
)

type ServiceLoadBalancerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete

func (r *ServiceLoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	service := &corev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, service); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return ctrl.Result{}, r.deleteManagedIpAddresses(ctx, service)
	}

	desiredIPs := serviceLoadBalancerIPs(service)
	if len(desiredIPs) == 0 {
		return ctrl.Result{}, r.deleteManagedIpAddresses(ctx, service)
	}

	desiredByAddress := make(map[string]struct{}, len(desiredIPs))
	for _, ip := range desiredIPs {
		desiredByAddress[ip] = struct{}{}
		if err := r.ensureIpAddressCR(ctx, service, ip); err != nil {
			return ctrl.Result{}, err
		}
	}

	existing := &netboxv1.IpAddressList{}
	if err := r.Client.List(ctx, existing, client.InNamespace(service.Namespace), client.MatchingLabels(serviceLoadBalancerLabels(service))); err != nil {
		return ctrl.Result{}, err
	}

	for i := range existing.Items {
		item := &existing.Items[i]
		if _, ok := desiredByAddress[item.Spec.IpAddress]; ok {
			continue
		}
		if err := r.Client.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		logger.Info("deleted stale ipaddress", "ipaddress", item.Name, "service", req.NamespacedName.String())
	}

	return ctrl.Result{}, nil
}

func (r *ServiceLoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Owns(&netboxv1.IpAddress{}).
		Complete(r)
}

func (r *ServiceLoadBalancerReconciler) ensureIpAddressCR(ctx context.Context, service *corev1.Service, ipAddress string) error {
	ipAddressResource := &netboxv1.IpAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceLoadBalancerIpAddressName(service.Name, ipAddress),
			Namespace: service.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ipAddressResource, func() error {
		ipAddressResource.Spec = netboxv1.IpAddressSpec{
			IpAddress:        ipAddress,
			Description:      fmt.Sprintf("Service %s/%s", service.Namespace, service.Name),
			PreserveInNetbox: false,
		}
		if ipAddressResource.Labels == nil {
			ipAddressResource.Labels = map[string]string{}
		}
		for key, value := range serviceLoadBalancerLabels(service) {
			ipAddressResource.Labels[key] = value
		}
		return controllerutil.SetControllerReference(service, ipAddressResource, r.Scheme)
	})
	return err
}

func (r *ServiceLoadBalancerReconciler) deleteManagedIpAddresses(ctx context.Context, service *corev1.Service) error {
	list := &netboxv1.IpAddressList{}
	if err := r.Client.List(ctx, list, client.InNamespace(service.Namespace), client.MatchingLabels(serviceLoadBalancerLabels(service))); err != nil {
		return err
	}
	for i := range list.Items {
		item := &list.Items[i]
		if err := r.Client.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func serviceLoadBalancerIPs(service *corev1.Service) []string {
	var ips []string
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.IP == "" {
			continue
		}
		ips = append(ips, normalizeIpAddress(ingress.IP))
	}
	return ips
}

func normalizeIpAddress(ip string) string {
	if strings.Contains(ip, "/") {
		return ip
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if parsed.To4() != nil {
		return ip + "/32"
	}
	return ip + "/128"
}

func serviceLoadBalancerLabels(service *corev1.Service) map[string]string {
	return map[string]string{
		serviceLoadBalancerManagedByLabel:        serviceLoadBalancerManagedByValue,
		serviceLoadBalancerServiceUIDLabel:       string(service.UID),
		serviceLoadBalancerServiceNameLabel:      service.Name,
		serviceLoadBalancerServiceNamespaceLabel: service.Namespace,
	}
}

func serviceLoadBalancerIpAddressName(serviceName, ipAddress string) string {
	hash := sha1.Sum([]byte(ipAddress))
	return fmt.Sprintf("%s-%s", serviceName, hex.EncodeToString(hash[:4]))
}
