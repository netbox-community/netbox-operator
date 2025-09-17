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

const IpRangeClaimFinalizerName = "iprangeclaim.netbox.dev/finalizer"

// IpRangeClaimReconciler reconciles a IpRangeClaim object
type IpRangeClaimReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpRangeClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpRangeClaim{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ipRange := &netboxv1.IpRange{}
	ipRangeName := o.Name
	ipRangeLookupKey := types.NamespacedName{
		Name:      ipRangeName,
		Namespace: o.Namespace,
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		err = r.Client.Get(ctx, ipRangeLookupKey, ipRange)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, removeFinalizer(ctx, r.Client, o, IpRangeClaimFinalizerName)
		}

		err = r.Client.Delete(ctx, ipRange)
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// requeue if owned iprange was still found
		return ctrl.Result{Requeue: true}, nil
	}

	// Set ready to false initially
	if apismeta.FindStatusCondition(o.Status.Conditions, netboxv1.ConditionReadyFalseNewResource.Type) == nil {
		err := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionReadyFalseNewResource, corev1.EventTypeNormal, nil)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialise Ready condition: %w, ", err)
		}
	}

	err = r.Client.Get(ctx, ipRangeLookupKey, ipRange)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.V(4).Info("iprange object matching iprange claim was not found, creating new iprange object")

		ipRangeModel, res, err := r.restoreOrAssignIpRangeAndSetCondition(ctx, o)
		if ipRangeModel == nil {
			return res, err
		}

		// create the IpRange CR
		ipRangeResource := generateIpRangeFromIpRangeClaim(ctx, o, ipRangeModel.StartAddress, ipRangeModel.EndAddress)
		err = controllerutil.SetControllerReference(o, ipRangeResource, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = addFinalizer(ctx, r.Client, o, IpRangeClaimFinalizerName)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Client.Create(ctx, ipRangeResource)
		if err != nil {
			errSetCondition := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, err)
			if errSetCondition != nil {
				return ctrl.Result{}, errSetCondition
			}
			return ctrl.Result{}, err
		}

		err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeAssignedTrue, corev1.EventTypeNormal,
			nil, fmt.Sprintf(" , assigned ip range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
		if err != nil {
			return ctrl.Result{}, err
		}

	} else {
		// update spec of IpRange object
		logger.V(4).Info("update iprange resource")
		ipRange.Spec = generateIpRangeSpec(o, ipRange.Spec.StartAddress, ipRange.Spec.EndAddress, logger)
		err = controllerutil.SetControllerReference(o, ipRange, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Client.Update(ctx, ipRange)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !apismeta.IsStatusConditionTrue(ipRange.Status.Conditions, "Ready") {
		err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeClaimReadyFalse, corev1.EventTypeWarning, nil)
		if err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("reconcile loop finished")
		return ctrl.Result{Requeue: true}, nil
	}

	logger.V(4).Info("iprange status ready true")
	o.Status, err = r.generateIpRangeClaimStatus(o, ipRange)
	if err != nil {
		logger.Error(err, "failed to generate ip range status")
		err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeClaimReadyFalseStatusGen, corev1.EventTypeWarning, err)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	err = r.Client.Status().Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeClaimReadyTrue, corev1.EventTypeNormal, nil)
	if err != nil {
		return ctrl.Result{}, err

	}
	logger.Info("reconcile loop finished")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpRangeClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpRangeClaim{}).
		Owns(&netboxv1.IpRange{}).
		Complete(r)
}

func (r *IpRangeClaimReconciler) tryLockOnParentPrefix(ctx context.Context, o *netboxv1.IpRangeClaim) (*leaselocker.LeaseLocker, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	leaseLockerNSN := types.NamespacedName{
		Name:      convertCIDRToLeaseLockName(o.Spec.ParentPrefix),
		Namespace: r.OperatorNamespace,
	}

	claimNSN := types.NamespacedName{
		Name:      o.ObjectMeta.Name,
		Namespace: o.Namespace,
	}

	ll, err := leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, claimNSN.String())
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	lockCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// try to lock lease for parent prefix
	locked := ll.TryLock(lockCtx)
	if !locked {
		// lock for parent prefix was not available, rescheduling
		logger.Info(fmt.Sprintf("failed to lock parent prefix %s", o.Spec.ParentPrefix))
		r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s",
			o.Spec.ParentPrefix)
		return nil, ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	logger.V(4).Info(fmt.Sprintf("successfully locked parent prefix %s", o.Spec.ParentPrefix))

	return ll, ctrl.Result{}, nil
}

func (r *IpRangeClaimReconciler) generateIpRangeClaimStatus(o *netboxv1.IpRangeClaim, ipRange *netboxv1.IpRange) (netboxv1.IpRangeClaimStatus, error) {
	startAddressDotDecimal := strings.Split(ipRange.Spec.StartAddress, "/")[0]
	endAddressDotDecimal := strings.Split(ipRange.Spec.EndAddress, "/")[0]

	availableIps, err := r.NetboxClient.GetAvailableIpAddressesByIpRange(ipRange.Status.IpRangeId)
	if err != nil {
		return netboxv1.IpRangeClaimStatus{}, err
	}

	ipAddresses := []string{}
	ipAddressesDotDecimal := []string{}
	for _, ip := range availableIps.Payload {
		ipAddresses = append(ipAddresses, ip.Address)
		ipAddressesDotDecimal = append(ipAddressesDotDecimal, strings.Split(ip.Address, "/")[0])
	}

	return netboxv1.IpRangeClaimStatus{
		IpRange:                fmt.Sprintf("%s-%s", ipRange.Spec.StartAddress, ipRange.Spec.EndAddress),
		IpRangeDotDecimal:      fmt.Sprintf("%s-%s", startAddressDotDecimal, endAddressDotDecimal),
		IpAddresses:            ipAddresses,
		IpAddressesDotDecimal:  ipAddressesDotDecimal,
		StartAddress:           ipRange.Spec.StartAddress,
		StartAddressDotDecimal: startAddressDotDecimal,
		EndAddress:             ipRange.Spec.EndAddress,
		EndAddressDotDecimal:   endAddressDotDecimal,
		IpRangeName:            ipRange.Name,
		Conditions:             o.Status.Conditions,
	}, nil
}

func (r *IpRangeClaimReconciler) restoreOrAssignIpRangeAndSetCondition(ctx context.Context, o *netboxv1.IpRangeClaim) (*models.IpRange, ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ll, res, err := r.tryLockOnParentPrefix(ctx, o)
	if err != nil {
		return nil, res, err
	}

	h := generateIpRangeRestorationHash(o)
	ipRangeModel, err := r.NetboxClient.RestoreExistingIpRangeByHash(h)
	if err != nil {
		if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
			return nil, ctrl.Result{}, errReport
		}
		return nil, ctrl.Result{Requeue: true}, nil
	}

	if ipRangeModel == nil {
		// ip range cannot be restored from netbox
		// assign new available ip range
		ipRangeModel, err = r.NetboxClient.GetAvailableIpRangeByClaim(
			&models.IpRangeClaim{
				ParentPrefix: o.Spec.ParentPrefix,
				Size:         o.Spec.Size,
				Metadata: &models.NetboxMetadata{
					Tenant: o.Spec.Tenant,
					Tags:   convertAPITagsToModelTags(o.Spec.Tags),
				},
			},
		)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, err); errReport != nil {
				return nil, ctrl.Result{}, errReport
			}
			return nil, ctrl.Result{Requeue: true}, nil
		}
		logger.V(4).Info(fmt.Sprintf("ip range is not reserved in netbox, assigned new ip range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
	} else {
		// reassign reserved ip range from netbox

		// check if the restored ip range has the size requested by the claim
		availableIpRanges, err := r.NetboxClient.GetAvailableIpAddressesByIpRange(ipRangeModel.Id)
		if err != nil {
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeClaimReadyFalse, corev1.EventTypeWarning, err, "failed getting available IP Addresses By Range"); errReport != nil {
				return nil, ctrl.Result{}, errReport
			}
			return nil, ctrl.Result{}, err
		}
		if len(availableIpRanges.Payload) != o.Spec.Size {
			ll.Unlock()
			err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionIpRangeAssignedFalseSizeMismatch, corev1.EventTypeWarning, err)
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			return nil, ctrl.Result{Requeue: true}, nil
		}
		logger.V(4).Info(fmt.Sprintf("reassign reserved ip range from netbox, range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
	}
	return ipRangeModel, ctrl.Result{}, nil
}
