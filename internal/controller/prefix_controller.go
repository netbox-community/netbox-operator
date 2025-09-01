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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/pkg/netbox/api"
	"github.com/swisscom/leaselocker"
)

const PrefixFinalizerName = "prefix.netbox.dev/finalizer"
const PXManagedCustomFieldsAnnotationName = "prefix.netbox.dev/managed-custom-fields"

// PrefixReconciler reconciles a Prefix object
type PrefixReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NetboxClient        *api.NetboxClient
	EventStatusRecorder *EventStatusRecorder
	OperatorNamespace   string
	RestConfig          *rest.Config
}

//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netbox.dev,resources=prefixes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PrefixReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	debugLogger := logger.V(4)

	logger.Info("reconcile loop started")

	/* 0. check if the matching Prefix object exists */
	o := &netboxv1.Prefix{}
	if err := r.Client.Get(ctx, req.NamespacedName, o); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// if being deleted
	if !o.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(o, PrefixFinalizerName) {
			if !o.Spec.PreserveInNetbox {
				if err := r.NetboxClient.DeletePrefix(o.Status.PrefixId); err != nil {
					if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyFalseDeletionFailed, corev1.EventTypeWarning, err); errReport != nil {
						return ctrl.Result{}, errReport
					}

					return ctrl.Result{Requeue: true}, nil
				}
			}

			debugLogger.Info("removing the finalizer")
			if removed := controllerutil.RemoveFinalizer(o, PrefixFinalizerName); !removed {
				return ctrl.Result{}, errors.New("failed to remove the finalizer")
			}

			if err := r.Update(ctx, o); err != nil {
				return ctrl.Result{}, err
			}
		}

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

	// register finalizer if not yet registered
	if !o.Spec.PreserveInNetbox && !controllerutil.ContainsFinalizer(o, PrefixFinalizerName) {
		debugLogger.Info("adding the finalizer")
		controllerutil.AddFinalizer(o, PrefixFinalizerName)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	/*
		1. try to lock the lease of the parent prefix if all of the following conditions are met
			- the prefix is owned by at least 1 prefixClaim
			- the prefix status condition is not ready
	*/
	ownerReferences := o.ObjectMeta.OwnerReferences
	var ll *leaselocker.LeaseLocker
	var err error
	if len(ownerReferences) > 0 /* len(nil array) = 0 */ && !apismeta.IsStatusConditionTrue(o.Status.Conditions, "Ready") {
		// get prefixClaim
		ownerReferencesLookupKey := types.NamespacedName{
			Name:      ownerReferences[0].Name, // TODO(henrybear327): Under what condition would we have more than 1 ownerReferences? What should we do with it?
			Namespace: req.Namespace,
		}
		prefixClaim := &netboxv1.PrefixClaim{}
		if err := r.Client.Get(ctx, ownerReferencesLookupKey, prefixClaim); err != nil {
			return ctrl.Result{}, err
		}

		if prefixClaim.Status.SelectedParentPrefix == "" {
			// the parent prefix is not selected
			if errReport := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyFalse, corev1.EventTypeWarning, fmt.Errorf("%s", "the parent prefix is not selected")); errReport != nil {
				return ctrl.Result{}, errReport
			}
			return ctrl.Result{
				Requeue: true,
			}, nil
		}

		if prefixClaim.Status.SelectedParentPrefix != msgCanNotInferParentPrefix {
			// we can't restore from the restoration hash

			// get the name of the parent prefix
			leaseLockerNSN := types.NamespacedName{
				Name:      convertCIDRToLeaseLockName(prefixClaim.Status.SelectedParentPrefix),
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
				errorMsg := fmt.Sprintf("failed to lock parent prefix %s", prefixClaim.Status.SelectedParentPrefix)
				r.EventStatusRecorder.Recorder().Eventf(o, corev1.EventTypeWarning, "FailedToLockParentPrefix", errorMsg)
				return ctrl.Result{
					RequeueAfter: 2 * time.Second,
				}, nil
			}
			debugLogger.Info("successfully locked parent prefix %s", prefixClaim.Status.SelectedParentPrefix)
		}
	}

	/* 2. reserve or update Prefix in netbox */
	accessor := apismeta.NewAccessor()
	annotations, err := accessor.Annotations(o)
	if err != nil {
		return ctrl.Result{}, err
	}

	prefixModel, err := generateNetboxPrefixModelFromPrefixSpec(&o.Spec, req, annotations[PXManagedCustomFieldsAnnotationName])
	if err != nil {
		return ctrl.Result{}, err
	}

	netboxPrefixModel, err := r.NetboxClient.ReserveOrUpdatePrefix(prefixModel)
	if err != nil {
		if errors.Is(err, api.ErrRestorationHashMismatch) && o.Status.PrefixId == 0 {
			// if there is a restoration hash mismatch and the PrefixId status field is not set,
			// delete the prefix so it can be recreated by the prefix claim controller
			// this will only affect resources that are created by a claim controller (and have a restoration hash custom field
			logger.Info("restoration hash mismatch, deleting prefix custom resource", "prefix", o.Spec.Prefix)
			err = r.Client.Delete(ctx, o)
			if err != nil {
				if updateStatusErr := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyFalse,
					corev1.EventTypeWarning, err); updateStatusErr != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update prefix status: %w, "+
						"after deletion of prefix cr failed: %w", updateStatusErr, err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, nil
		}

		if updateStatusErr := r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyFalse,
			corev1.EventTypeWarning, err); updateStatusErr != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update prefix status: %w, "+
				"after reservation of prefix netbox failed: %w", updateStatusErr, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	/* 3. unlock lease of parent prefix */
	if ll != nil {
		ll.UnlockWithRetry(ctx)
	}

	/* 4. update status fields */
	o.Status.PrefixId = netboxPrefixModel.ID
	o.Status.PrefixUrl = config.GetBaseUrl() + "/ipam/prefixes/" + strconv.FormatInt(netboxPrefixModel.ID, 10)
	err = r.Client.Status().Update(ctx, o)
	if err != nil {
		return ctrl.Result{}, err
	}

	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[PXManagedCustomFieldsAnnotationName], err = generateManagedCustomFieldsAnnotation(o.Spec.CustomFields)
	if err != nil {
		logger.Error(err, "failed to update last metadata annotation")
		return ctrl.Result{Requeue: true}, nil
	}

	err = accessor.SetAnnotations(o, annotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update object to store lastIpAddressMetadata annotation
	if err := r.Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	// check if the created prefix contains the entire description from spec
	if _, found := strings.CutPrefix(netboxPrefixModel.Description, req.NamespacedName.String()+" // "+o.Spec.Description); !found {
		r.EventStatusRecorder.Recorder().Event(o, corev1.EventTypeWarning, "PrefixDescriptionTruncated", "prefix was created with truncated description")
	}

	debugLogger.Info(fmt.Sprintf("reserved prefix in netbox, prefix: %s", o.Spec.Prefix))
	if err = r.EventStatusRecorder.Report(ctx, o, netboxv1.ConditionPrefixReadyTrue, corev1.EventTypeNormal, nil); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconcile loop finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrefixReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netboxv1.Prefix{}).
		Complete(r)
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

	convertedTags := make([]models.Tag, len(spec.Tags))
	for i, t := range spec.Tags {
		convertedTags[i] = models.Tag{
			Name: t.Name,
			Slug: t.Slug,
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
			Tags:        convertedTags,
		},
	}, nil
}
