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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IpRangeClaimReconciler reconciles a IpRangeClaim object
type IpRangeClaimReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	NetboxClient      *api.NetboxClient
	Recorder          record.EventRecorder
	OperatorNamespace string
	RestConfig        *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=iprangeclaims/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpRangeClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching IpRangeClaim object exists */
	o := &netboxv1.IpRangeClaim{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// 1. check if matching IpRange object already exists
	ipRange := &netboxv1.IpRange{}
	ipRangeName := o.ObjectMeta.Name
	ipRangeLookupKey := types.NamespacedName{
		Name:      ipRangeName,
		Namespace: o.Namespace,
	}

	err = r.Client.Get(ctx, ipRangeLookupKey, ipRange)
	if err != nil {
		// return error if not a notfound error
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		debugLogger.Info("iprange object matching iprange claim was not found, creating new iprange object")

		// 2. check if lease for parent prefix is available
		ll, res, err := r.tryLockOnParentPrefix(ctx, o)
		if err != nil {
			return res, err
		}

		// 4. try to reclaim ip range
		h := generateIpRangeRestorationHash(o)
		ipRangeModel, err := r.NetboxClient.RestoreExistingIpRangeByHash(h)
		if err != nil {
			if err := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, "", err); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if ipRangeModel == nil {
			// ip range cannot be restored from netbox
			// 5.a assign new available ip range
			ipRangeModel, err = r.NetboxClient.GetAvailableIpRangeByClaim(
				&models.IpRangeClaim{
					ParentPrefix: o.Spec.ParentPrefix,
					Size:         o.Spec.Size,
					Metadata: &models.NetboxMetadata{
						Tenant: o.Spec.Tenant,
					},
				},
			)
			if err != nil {
				if err := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, "", err); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			debugLogger.Info(fmt.Sprintf("ip range is not reserved in netbox, assigned new ip range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
		} else {
			// 5.b reassign reserved ip range from netbox

			// check if the restored ip range has the size requested by the claim
			availableIps, err := r.NetboxClient.GetAvailableIpAddressesByIpRange(ipRangeModel.Id)
			if len(availableIps.Payload) != o.Spec.Size {
				ll.Unlock()
				err = r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeAssignedFalseSizeMissmatch, corev1.EventTypeWarning, "", err)
				if err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			debugLogger.Info(fmt.Sprintf("reassign reserved ip range from netbox, range: %s-%s", ipRangeModel.StartAddress, ipRangeModel.EndAddress))
		}

		// 6.a create the IpRange CR
		err = r.createIpRangeCR(ctx, o, ipRangeModel)
		if err != nil {
			return ctrl.Result{}, err
		}

	} else {
		// 6.b update fields of IPRange object
		debugLogger.Info("update iprange resource")
		err := r.Client.Get(ctx, ipRangeLookupKey, ipRange)
		if err != nil {
			return ctrl.Result{}, err
		}

		updatedIpRangeSpec := generateIpRangeSpec(o, ipRange.Spec.StartAddress, ipRange.Spec.EndAddress, logger)
		_, err = ctrl.CreateOrUpdate(ctx, r.Client, ipRange, func() error {
			// only add the mutable fields here
			ipRange.Spec.CustomFields = updatedIpRangeSpec.CustomFields
			ipRange.Spec.Comments = updatedIpRangeSpec.Comments
			ipRange.Spec.Description = updatedIpRangeSpec.Description
			ipRange.Spec.PreserveInNetbox = updatedIpRangeSpec.PreserveInNetbox
			return nil
		})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = r.updateIpRangeClaimStatus(ctx, o, ipRange)
	if err != nil {
		return ctrl.Result{}, err
	}

	if apismeta.IsStatusConditionFalse(o.Status.Conditions, "Ready") {
		return ctrl.Result{Requeue: true}, nil
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

// SetConditionAndCreateEvent updates the condition and creates a log entry and event for this condition change
func (r *IpRangeClaimReconciler) SetConditionAndCreateEvent(ctx context.Context, o *netboxv1.IpRangeClaim, condition metav1.Condition, eventType string, conditionMessageAppend string, errIn error) error {
	if len(conditionMessageAppend) > 0 {
		condition.Message = condition.Message + ". " + conditionMessageAppend
	}

	if errIn != nil {
		// log errors here, so we do not need to aggregate them in the reconcile loop with the error of updating the status
		logger := log.FromContext(ctx)
		logger.Error(errIn, condition.Message+errIn.Error())
		condition.Message = condition.Message + ". Check the logs for more information."
	}

	conditionChanged := apismeta.SetStatusCondition(&o.Status.Conditions, condition)
	if conditionChanged {
		r.Recorder.Event(o, eventType, condition.Reason, condition.Message)
	}

	err := r.Client.Status().Update(ctx, o)
	if err != nil {
		return err
	}

	return nil
}

func (r *IpRangeClaimReconciler) tryLockOnParentPrefix(ctx context.Context, o *netboxv1.IpRangeClaim) (*leaselocker.LeaseLocker, ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

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

	// 3. try to lock lease for parent prefix
	locked := ll.TryLock(lockCtx)
	if !locked {
		// lock for parent prefix was not available, rescheduling
		logger.Info(fmt.Sprintf("failed to lock parent prefix %s", o.Spec.ParentPrefix))
		r.Recorder.Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s",
			o.Spec.ParentPrefix)
		return nil, ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", o.Spec.ParentPrefix))

	return ll, ctrl.Result{}, nil
}

func (r *IpRangeClaimReconciler) updateIpRangeClaimStatus(ctx context.Context, o *netboxv1.IpRangeClaim, ipRange *netboxv1.IpRange) error {
	debugLogger := log.FromContext(ctx).V(4)

	if apismeta.IsStatusConditionTrue(ipRange.Status.Conditions, "Ready") {
		debugLogger.Info("iprange status ready true")

		startAddressDotDecimal := strings.Split(ipRange.Spec.StartAddress, "/")[0]
		endAddressDotDecimal := strings.Split(ipRange.Spec.EndAddress, "/")[0]

		o.Status.IpRange = fmt.Sprintf("%s-%s", ipRange.Spec.StartAddress, ipRange.Spec.EndAddress)

		o.Status.IpRangeDotDecimal = fmt.Sprintf("%s-%s", startAddressDotDecimal, endAddressDotDecimal)

		availableIps, err := r.NetboxClient.GetAvailableIpAddressesByIpRange(ipRange.Status.IpRangeId)
		if err != nil {
			return err
		}

		o.Status.IpAddresses = []string{}
		o.Status.IpAddressesDotDecimal = []string{}
		for _, ip := range availableIps.Payload {
			o.Status.IpAddresses = append(o.Status.IpAddresses, ip.Address)
			o.Status.IpAddressesDotDecimal = append(o.Status.IpAddressesDotDecimal, strings.Split(ip.Address, "/")[0])
		}

		o.Status.StartAddress = ipRange.Spec.StartAddress
		o.Status.StartAddressDotDecimal = startAddressDotDecimal

		o.Status.EndAddress = ipRange.Spec.EndAddress
		o.Status.EndAddressDotDecimal = endAddressDotDecimal

		o.Status.IpRangeName = ipRange.Name

		err = r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeReadyTrue, corev1.EventTypeNormal, "", nil)
		if err != nil {
			return err
		}
	} else {
		debugLogger.Info("iprange status ready false")
		err := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeReadyFalse, corev1.EventTypeWarning, "", nil)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (r *IpRangeClaimReconciler) createIpRangeCR(ctx context.Context, o *netboxv1.IpRangeClaim, ipRangeModel *models.IpRange) error {
	ipRangeResource := generateIpRangeFromIpRangeClaim(ctx, o, ipRangeModel.StartAddress, ipRangeModel.EndAddress)
	err := controllerutil.SetControllerReference(o, ipRangeResource, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Client.Create(ctx, ipRangeResource)
	if err != nil {
		err := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeAssignedFalse, corev1.EventTypeWarning, "", err)
		if err != nil {
			return err
		}
		return err
	}

	err = r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeAssignedTrue, corev1.EventTypeNormal, "", nil)
	if err != nil {
		return err
	}

	return nil
}
