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

	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/models"
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

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/swisscom/leaselocker"
)

const PrefixFinalizerName = "prefix.netbox.dev/finalizer"
const LastPrefixMetadataAnnotationName = "prefix.netbox.dev/last-prefix-metadata"

// PrefixReconciler reconciles a Prefix object
type PrefixReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	NetboxClient      *api.NetboxClient
	Recorder          record.EventRecorder
	OperatorNamespace string
	RestConfig        *rest.Config
}

// +kubebuilder:rbac:groups=netbox.dev,resources=prefixes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=netbox.dev,resources=prefixes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=netbox.dev,resources=prefixes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PrefixReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("prefix reconcile loop started")

	/* 0. check if the matching Prefix object exists */
	prefix := &netboxv1.Prefix{}
	if err := r.Client.Get(ctx, req.NamespacedName, prefix); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !prefix.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(prefix, PrefixFinalizerName) {
			if !prefix.Spec.PreserveInNetbox {
				if err := r.NetboxClient.DeletePrefix(prefix.Status.PrefixId); err != nil {
					return ctrl.Result{}, err
				}
			}

			debugLogger.Info("removing the finalizer")
			if removed := controllerutil.RemoveFinalizer(prefix, PrefixFinalizerName); !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			if err := r.Update(ctx, prefix); err != nil {
				return ctrl.Result{}, err
			}
		}

		// end loop if deletion timestamp is not zero
		return ctrl.Result{}, nil
	}

	/*
		1. try to lock the lease of the parent prefix if all of the following conditions are met
			- the prefix is owned by at least 1 prefixClaim
			- the prefix status condition is not ready
	*/
	ownerReferences := prefix.ObjectMeta.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var err error
	if len(ownerReferences) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(prefix.Status.Conditions, "Ready") {
		// get prefixClaim
		ownerReferencesLookupKey := types.NamespacedName{
			Name:      ownerReferences[0].Name, // TODO(henrybear327): Under what condition would we have more than 1 ownerReferences? What should we do with it?
			Namespace: req.Namespace,
		}
		prefixClaim := &netboxv1.PrefixClaim{}
		if err := r.Client.Get(ctx, ownerReferencesLookupKey, prefixClaim); err != nil {
			return ctrl.Result{}, err
		}

		// get the name of the parent prefix
		parentPrefixName := strings.Replace(prefixClaim.Spec.ParentPrefix, "/", "-", -1)

		leaseLockerNSN := types.NamespacedName{Name: parentPrefixName, Namespace: r.OperatorNamespace}
		ll, err = leaselocker.NewLeaseLocker(r.RestConfig, leaseLockerNSN, req.NamespacedName.String())
		if err != nil {
			return ctrl.Result{}, err
		}

		lockCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// create lock
		if locked := ll.TryLock(lockCtx); !locked {
			logger.Info(fmt.Sprintf("failed to lock parent prefix %s", parentPrefixName))
			r.Recorder.Eventf(prefix, corev1.EventTypeWarning, "FailedToLockParentPrefix", "failed to lock parent prefix %s", parentPrefixName)
			return ctrl.Result{
				RequeueAfter: 2 * time.Second,
			}, nil
		}
		debugLogger.Info("sucessfully locked parent prefix %s", parentPrefixName)
	}

	/* 2. reserve or update Prefix in netbox */
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(prefix)
	if err != nil {
		return ctrl.Result{}, err
	}

	prefixModel, err := generateNetboxPrefixModelFromPrefixSpec(&prefix.Spec, req, annotations[LastPrefixMetadataAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxPrefixModel, err := r.NetboxClient.ReserveOrUpdatePrefix(prefixModel)
	if err != nil {
		updateStatusErr := r.SetConditionAndCreateEvent(ctx, prefix, netboxv1.ConditionPrefixReadyFalse, corev1.EventTypeWarning, prefix.Spec.Prefix)
		return ctrl.Result{}, fmt.Errorf("failed at update prefix status: %w, "+"after reservation of prefix in netbox failed: %w", updateStatusErr, err)
	}

	// update lastPrefixMetadata annotation
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if len(prefix.Spec.CustomFields) > 0 {
		lastPrefixMetadata, err := json.Marshal(prefix.Spec.CustomFields)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal lastPrefixMetadata annotation: %w", err)
		}
		annotations[LastPrefixMetadataAnnotationName] = string(lastPrefixMetadata)
	} else {
		annotations[LastPrefixMetadataAnnotationName] = "{}"
	}
	err = accessor.SetAnnotations(prefix, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update object to store lastIpAddressMetadata annotation
	if err := r.Update(ctx, prefix); err != nil {
		return ctrl.Result{}, err
	}

	// check if the created prefix contains the entire description from spec
	if _, found := strings.CutPrefix(netboxPrefixModel.Description, req.NamespacedName.String()+" // "+prefix.Spec.Description); !found {
		r.Recorder.Event(prefix, corev1.EventTypeWarning, "PrefixDescriptionTruncated", "prefix was created with truncated description")
	}

	// register finalizer if not yet registered
	if !prefix.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(prefix, PrefixFinalizerName) {
		debugLogger.Info("adding the finalizer")
		controllerutil.AddFinalizer(prefix, PrefixFinalizerName)
		if err := r.Update(ctx, prefix); err != nil {
			return ctrl.Result{}, err
		}
	}

	debugLogger.Info(fmt.Sprintf("reserved prefix in netbox, prefix: %s", prefix.Spec.Prefix))

	/* 3. unlock lease of parent prefix */
	if ll != nil {
		ll.Unlock()
	}

	/* 4. update status conditions */
	prefix.Status.PrefixId = netboxPrefixModel.ID
	prefix.Status.PrefixUrl = config.GetBaseUrl() + "/ipam/prefixes/" + strconv.FormatInt(netboxPrefixModel.ID, 10)
	if err = r.SetConditionAndCreateEvent(ctx, prefix, netboxv1.ConditionPrefixReadyTrue, corev1.EventTypeNormal, ""); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("prefix reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrefixReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Prefix{}).
		Complete(r)
}

// TODO(henrybear327): Duplicated code, consider refactoring this
func (r *PrefixReconciler) SetConditionAndCreateEvent(ctx context.Context, o *netboxv1.Prefix, condition metav1.Condition, eventType string, conditionMessageAppend string) error {
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

func generateNetboxPrefixModelFromPrefixSpec(spec *netboxv1.PrefixSpec, req ctrl.Request, lastPrefixMetadata string) (*models.Prefix, error) {
	// unmarshal lastPrefixMetadata json string to map[string]string
	lastAppliedCustomFields := make(map[string]string)
	if lastPrefixMetadata != "" {
		if err := json.Unmarshal([]byte(lastPrefixMetadata), &lastAppliedCustomFields); err != nil {
			return nil, fmt.Errorf("failed to unmarshal lastPrefixMetadata annotation: %w", err)
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

	return &models.Prefix{
		Prefix: spec.Prefix,
		Metadata: &models.NetboxMetadata{
			Comments:    spec.Comments,
			Custom:      netboxCustomFields,
			Description: req.NamespacedName.String() + " // " + spec.Description,
			Site:        spec.Site,
			Tenant:      spec.Tenant,
		},
	}, nil
}
