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

const IpAddressFinalizerName = "ipaddress.netbox.dev/finalizer"
const IPManagedCustomFieldsAnnotationName = "ipaddress.netbox.dev/managed-custom-fields"

// IpAddressReconciler reconciles a IpAddress object
type IpAddressReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxCompositeClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=ipaddresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IpAddressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (reconcileResult ctrl.Result, reconcileErr error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started")

	o := &netboxv1.IpAddress{}

	err := r.Get(ctx, req.NamespacedName, o)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Snapshot for status patch — taken before any status mutations so the
	// merge-patch diff captures every change (IpAddressId, conditions, etc.).
	statusBase := o.DeepCopy()

	// cancelLock stops the lease renewal goroutine on early returns (lease expires naturally).
	// Explicit cancelLock()+UnlockWithRetry() runs inline after the critical section.
	var cancelLock context.CancelFunc

	// Defer status update to ensure it happens regardless of how we exit
	defer func() {
		reconcileResult, reconcileErr = r.updateStatus(ctx, o, statusBase, reconcileResult, reconcileErr)
	}()

	// if being deleted
	if !o.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
			return ctrl.Result{}, nil
		}

		if !o.Spec.PreserveInNetbox && o.Status.IpAddressId != 0 {
			if err = r.NetboxClient.DeleteIpAddress(o.Status.IpAddressId); err != nil {
				return ctrl.Result{}, NewDomainError("failed to delete ip address from netbox: %w", err)
			}
		}

		logger.V(4).Info("removing the finalizer")
		removed := controllerutil.RemoveFinalizer(o, IpAddressFinalizerName)
		if !removed {
			return ctrl.Result{}, errors.New("failed to remove the finalizer")
		}

		if err = r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// if PreserveIpInNetbox flag is false then register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, IpAddressFinalizerName) {
		logger.V(4).Info("adding the finalizer")
		controllerutil.AddFinalizer(o, IpAddressFinalizerName)
		if err = r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 1. try to lock lease of parent prefix if IpAddressUrl is not set in status
	// and IpAddress is owned by an IpAddressClaim
	or := o.OwnerReferences
	var ll *leaselocker.LeaseLocker
	if len(or) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get ip address claim
		orLookupKey := types.NamespacedName{
			Name:      or[0].Name,
			Namespace: req.Namespace,
		}
		ipAddressClaim := &netboxv1.IpAddressClaim{}
		err = r.Get(ctx, orLookupKey, ipAddressClaim)
		if err != nil {
			return ctrl.Result{}, err
		}

		// get name of parent prefix
		leaseLockerNSN := types.NamespacedName{
			Name:      convertCIDRToLeaseLockName(ipAddressClaim.Spec.ParentPrefix),
			Namespace: r.OperatorNamespace,
		}
		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.String())
		if err != nil {
			return ctrl.Result{}, err
		}

		var lockCtx context.Context
		lockCtx, cancelLock = context.WithTimeout(ctx, lockAcquireTimeout)
		defer func() {
			if cancelLock != nil {
				cancelLock() // ensure renewal goroutine stops on any return path
			}
		}()
		locked := ll.TryLock(lockCtx)
		if !locked {
			errorMsg := fmt.Sprintf("failed to lock parent prefix %s", ipAddressClaim.Spec.ParentPrefix)
			r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, NewDomainError("%s", errorMsg)
		}
		logger.V(4).Info("successfully locked parent prefix", "prefix", ipAddressClaim.Spec.ParentPrefix)
	}

	// 2. reserve or update ip address in netbox
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	ipAddressModel, err := generateNetboxIpAddressModelFromIpAddressSpec(&o.Spec, req, annotations[IPManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxIpAddressModel, skipsUpdate, err := r.NetboxClient.ReserveOrUpdateIpAddress(ipAddressModel, o)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.IpAddressId == 0 {
			// if there is a restoration hash mismatch and the IpAddressId status field is not set,
			// delete the ip address so it can be recreated by the ip address claim controller
			logger.Info("restoration hash mismatch, deleting ip address custom resource", "ipaddress", o.Spec.IpAddress)
			if deleteErr := r.Delete(ctx, o); deleteErr != nil {
				return ctrl.Result{}, NewDomainError("failed to delete IpAddress CR with restoration hash mismatch: %w", deleteErr)
			}
			// Object deleted - status update in deferred function will be ignored via client.IgnoreNotFound
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, NewDomainError("%w", err)
	}

	// 3. unlock lease of parent prefix — allocation is done, lock no longer needed
	if ll != nil {
		cancelLock()
		ll.UnlockWithRetry(ctx)
	}

	// 4. if no change in spec generation and NetBox object, skip K8s status update
	if skipsUpdate {
		return ctrl.Result{}, nil
	}

	// 4.1 update annotations
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[IPManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		return ctrl.Result{}, NewDomainError("failed to generate managed custom fields annotation: %w", err)
	}

	// snapshot before annotation mutation for merge-patch
	patch := client.MergeFrom(o.DeepCopy())

	if err = accessor.SetAnnotations(o, annotations); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Patch(ctx, o, patch); err != nil {
		return ctrl.Result{}, err
	}

	// 4. update status fields (set after r.Patch to avoid being overwritten by API response)
	o.Status.IpAddressId = netboxIpAddressModel.ID
	o.Status.IpAddressUrl = config.GetBaseUrl() + "/ipam/ip-addresses/" + strconv.FormatInt(netboxIpAddressModel.ID, 10)

	// check if created ip address contains entire description from spec
	_, found := strings.CutPrefix(netboxIpAddressModel.Description, req.String()+" // "+o.Spec.Description)
	if !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "IpDescriptionTruncated", "ip address was created with truncated description")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpAddressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.IpAddress{}).
		Complete(r)
}

// updateStatus updates the IpAddress status conditions based on the current state of the object.
// This function is called as a deferred function in Reconcile to ensure status is always updated.
// It captures any reconcile errors to include them in the status condition message.
func (r *IpAddressReconciler) updateStatus(ctx context.Context, o *netboxv1.IpAddress, statusBase *netboxv1.IpAddress, reconcileRes ctrl.Result, reconcileErr error) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	// Set default return values
	result = reconcileRes
	err = reconcileErr

	if apierrors.IsConflict(err) {
		// Object was modified concurrently — skip status update, will retry on requeue
		return IgnoreDomainError(result, err)
	}

	logger.V(4).Info("updating ipaddress status")

	switch {
	case !o.DeletionTimestamp.IsZero() && reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalseDeletionFailed, corev1.EventTypeWarning, reconcileErr)
	case !o.DeletionTimestamp.IsZero():
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalseDeletionInProgress, corev1.EventTypeNormal, nil)
	case o.Status.IpAddressUrl == "":
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalse, corev1.EventTypeWarning, reconcileErr)
	case reconcileErr != nil:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyFalse, corev1.EventTypeWarning, reconcileErr)
	default:
		r.EventStatusRecorder.Report(ctx, o,
			netboxv1.ConditionIpaddressReadyTrue, corev1.EventTypeNormal, nil)
	}

	// Align resource version so the patch targets the latest revision
	statusBase.SetResourceVersion(o.GetResourceVersion())
	statusPatch := client.MergeFrom(statusBase)
	patchErr := r.Status().Patch(ctx, o, statusPatch)
	if patchErr != nil {
		patchErr = client.IgnoreNotFound(patchErr)
		if patchErr != nil {
			err = errors.Join(err, patchErr)
		}
	}

	return IgnoreDomainError(result, err)
}

func generateNetboxIpAddressModelFromIpAddressSpec(spec *netboxv1.IpAddressSpec, req ctrl.Request, lastIpAddressMetadata string) (*models.IPAddress, error) {
	// unmarshal lastIpAddressMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastIpAddressMetadata != "" {
		if err := json.Unmarshal([]byte(lastIpAddressMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastIpAddressMetadata annotation: %w", err)
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

	return &models.IPAddress{
		IpAddress: spec.IpAddress,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.String() + " // " + spec.Description,
			Tenant:      spec.Tenant,
		},
	}, nil
}
