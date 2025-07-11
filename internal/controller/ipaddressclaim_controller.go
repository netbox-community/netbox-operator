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
	"fmt"
	"strings"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IpAddressClaimReconciler reconciles a IpAddressClaim object
type IpAddressClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddressclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpAddressClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching IpAddressClaim object exists */
	o := &netboxv1.IpAddressClaim{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialise Ready condition: %w, ", err)
		}
	}

	// 1. check if matching IpAddress object already exists
	ipAddress := &netboxv1.IpAddress{}
	ipAddressName := o.ObjectMeta.Name
	ipAddressLookupKey := types.NamespacedName{
		Name:      ipAddressName,
		Namespace: o.Namespace,
	}

	err = r.Client.Get(ctx, ipAddressLookupKey, ipAddress)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		debugLogger.Info("ipaddress object matching ipaddress claim was not found, creating new ipaddress object")

		// 2. check if lease for parent prefix is available
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(o.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.Namespace+"/"+ipAddressName)
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// 3. try to lock lease for parent prefix
		locked := ll.TryLock(lockCtx)
		if !locked {
			// lock for parent prefix was not available, rescheduling
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", o.Spec.ParentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", o.Spec.ParentPrefix))

		// 4. try to reclaim ip address
		h := generateIpAddressRestorationHash(o)
		ipAddressModel, err := r.NetboxClient.RestoreExistingIpByHash(h)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if ipAddressModel == nil {
			// ip address cannot be restored from netbox
			// 5.a assign new available ip address
			ipAddressModel, err = r.NetboxClient.GetAvailableIpAddressByClaim(
				&models.IPAddressClaim{
					ParentPrefix: o.Spec.ParentPrefix,
					Metadata: &models.NetboxMetadata{
						Tenant: o.Spec.Tenant,
					},
				})
			if err != nil {
				if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
					return ctrl.Result{}, errReport
				}
				return ctrl.Result{Requeue: true}, nil
			}
			debugLogger.Info(fmt.Sprintf("ip address is not reserved in netbox, assigned new ip address: %s", ipAddressModel.IpAddress))
		} else {
			// 5.b reassign reserved ip address from netbox
			// do nothing, ip address restored
			debugLogger.Info(fmt.Sprintf("reassign reserved ip address from netbox, ip: %s", ipAddressModel.IpAddress))
		}

		// 6.a create the IPAddress object
		ipAddressResource := generateIpAddressFromIpAddressClaim(o, ipAddressModel.IpAddress, logger)
		err = controllerutil.SetControllerReference(o, ipAddressResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Client.Create(ctx, ipAddressResource)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{}, err
		}

		err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpAssignedTrue, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, err
		}

	} else {
		// 6.b update fields of IPAddress object
		debugLogger.Info("update ipaddress resource")
		err := r.Client.Get(ctx, ipAddressLookupKey, ipAddress)
		if err != nil {
			return ctrl.Result{}, err
		}

		updatedIpAddressSpec := generateIpAddressSpec(o, ipAddress.Spec.IpAddress, logger)
		err = controllerutil.SetControllerReference(o, ipAddress, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, ipAddress, func() error {
			// only add the mutable fields here
			ipAddress.Spec.CustomFields = updatedIpAddressSpec.CustomFields
			ipAddress.Spec.Comments = updatedIpAddressSpec.Comments
			ipAddress.Spec.Description = updatedIpAddressSpec.Description
			ipAddress.Spec.PreserveInNetbox = updatedIpAddressSpec.PreserveInNetbox
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 7. update IPAddressClaim Ready status
	debugLogger.Info("update ipaddressclaim status")

	if apismeta.IsStatusConditionTrue(ipAddress.Status.Conditions, "Ready") {
		debugLogger.Info("ipaddress status ready true")
		o.Status.IpAddress = ipAddress.Spec.IpAddress
		o.Status.IpAddressDotDecimal = strings.Split(ipAddress.Spec.IpAddress, "/")[0]
		o.Status.IpAddressName = ipAddress.Name
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpClaimReadyTrue, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		debugLogger.Info("ipaddress status ready false")
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpClaimReadyFalse, corev1.EventTypeWarning, nil)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("reconcile loop finished")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpAddressClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpAddressClaim{}).
		Owns(&netboxv1.IpAddress{}).
		Complete(r)
}
