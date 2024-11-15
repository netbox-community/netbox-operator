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
	"maps"
	"strconv"
	"strings"
	"time"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
	"github.com/swisscom/leaselocker"
	corev1 "k8s.io/api/core/v1"
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

const IpRangeFinalizerName = "iprange.netbox.dev/finalizer"
const LastIpRangeMetadataAnnotationName = "iprange.netbox.dev/last-ip-range-metadata"

// IpRangeReconciler reconciles a IpRange object
type IpRangeReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	NetboxClient      *api.NetboxClient
	Recorder          record.EventRecorder
	OperatorNamespace string
	RestConfig        *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipranges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpRangeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpRange{}
	err := r.Client.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, IpRangeFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				err := r.NetboxClient.DeleteIpRange(o.Status.IpRangeId)
				if err != nil {
					setConditionErr := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeReadyFalseDeletionFailed, corev1.EventTypeWarning, err.Error())
					if setConditionErr != nil {
						return ctrl.Result{}, fmt.Errorf("error updating status: %w, when deletion of IpRange failed: %w", setConditionErr, err)
					}

					return ctrl.Result{Requeue: true}, nil
				}
			}

			debugLogger.Info("removing the finalizer")
			removed := controllerutil.RemoveFinalizer(o, IpRangeFinalizerName)
			if !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			err = r.Update(ctx, o)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	// if PreserveIpInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, IpRangeFinalizerName) {
		debugLogger.Info("adding the finalizer")
		controllerutil.AddFinalizer(o, IpRangeFinalizerName)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of parent prefix if IpRange status condition is not true
	// and IpRange is owned by an IpRangeClaim
	or := o.ObjectMeta.OwnerReferences
	var ll *leaselocker.LeaseLocker
	if len(or) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get ip range claim
		orLookupKey := types.NamespacedName{
			Name:      or[0].Name,
			Namespace: req.Namespace,
		}
		ipRangeClaim := &netboxv1.IpRangeClaim{}
		err = r.Client.Get(ctx, orLookupKey, ipRangeClaim)
		if err != nil {
			return ctrl.Result{}, err
		}

		// get name of parent prefix
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(ipRangeClaim.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.NamespacedName.String())
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// create lock
		locked := ll.TryLock(lockCtx)
		if !locked {
			logger.Info(fmt.Sprintf("failed to lock parent prefix %s", ipRangeClaim.Spec.ParentPrefix))
			r.Recorder.Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s",
				ipRangeClaim.Spec.ParentPrefix)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info(fmt.Sprintf("successfully locked parent prefix %s", ipRangeClaim.Spec.ParentPrefix))
	}

	// 2. reserve or update ip range in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	ipRangeModel, err := generateNetboxIpRangeModelFromIpRangeSpec(&o.Spec, req, annotations[LastIpRangeMetadataAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxIpRangeModel, err := r.NetboxClient.ReserveOrUpdateIpRange(ipRangeModel)
	if err != nil {
		updateStatusErr := r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeReadyFalse,
			corev1.EventTypeWarning, fmt.Sprintf("%s-%s", o.Spec.StartAddress, o.Spec.EndAddress))
		return ctrl.Result{}, fmt.Errorf("failed to update ip range status: %w, "+
			"after reservation of ip in netbox failed: %w", updateStatusErr, err)
	}

	// 3. unlock lease of parent prefix
	if ll != nil {
		ll.Unlock()
	}

	// 4. update status fields
	o.Status.IpRangeId = netboxIpRangeModel.ID
	o.Status.IpRangeUrl = config.GetBaseUrl() + "/ipam/ip-rages/" + strconv.FormatInt(netboxIpRangeModel.ID, 10)
	err = r.Client.Status().Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update lastIpRangeMetadata annotation
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if len(o.Spec.CustomFields) > 0 {
		lastIpRangeMetadata, err := json.Marshal(o.Spec.CustomFields)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal lastIpRangeMetadata annotation: %w", err)
		}

		annotations[LastIpRangeMetadataAnnotationName] = string(lastIpRangeMetadata)
	} else {
		annotations[LastIpRangeMetadataAnnotationName] = "{}"
	}

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update object to store lastIpRangeMetadata annotation
	if err := r.Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	// check if created ip range contains entire description from spec
	_, found := strings.CutPrefix(netboxIpRangeModel.Description, req.NamespacedName.String()+" // "+o.Spec.Description)
	if !found {
		r.Recorder.Event(o, corev1.EventTypeWarning, "IpRangeDescriptionTruncated", "ip range was created with truncated description")
	}

	debugLogger.Info(fmt.Sprintf("reserved ip range in netbox, start address: %s, end address: %s", o.Spec.StartAddress, o.Spec.EndAddress))

	err = r.SetConditionAndCreateEvent(ctx, o, netboxv1.ConditionIpRangeReadyTrue, corev1.EventTypeNormal, "")
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpRangeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpRange{}).
		Complete(r)
}

func (r *IpRangeReconciler) SetConditionAndCreateEvent(ctx context.Context, o *netboxv1.IpRange, condition metav1.Condition, eventType string, conditionMessageAppend string) error {
	if len(conditionMessageAppend) > 0 {
		condition.Message = condition.Message + ". " + conditionMessageAppend
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

func generateNetboxIpRangeModelFromIpRangeSpec(spec *netboxv1.IpRangeSpec, req ctrl.Request, lastIpRangeMetadata string) (*models.IpRange, error) {
	// unmarshal lastIpRangeMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastIpRangeMetadata != "" {
		if err := json.Unmarshal([]byte(lastIpRangeMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastIpRangeMetadata annotation: %w", err)
		}
	}

	netboxCustomFields := make(map[string]string)
	if len(spec.CustomFields) > 0 {
		netboxCustomFields = maps.Clone(spec.CustomFields)
	}

	// if a custom field was removed from the spec, add it with an empty value

	for key := range lastAppliedCustomFields {
		_, ok := netboxCustomFields[key]
		if !ok {
			netboxCustomFields[key] = ""
		}
	}

	return &models.IpRange{
		StartAddress: spec.StartAddress,
		EndAddress:   spec.EndAddress,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.NamespacedName.String() + " // " + spec.Description,
			Tenant:      spec.Tenant,
		},
	}, nil
}
