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
	"net/http"
	"sync"
	"time"

	v4client "github.com/netbox-community/go-netbox/v4"
	netboxv1 "github.com/netbox-community/netbox-operator/api/v1"
	"github.com/netbox-community/netbox-operator/gen/mock_interfaces"
	"github.com/netbox-community/netbox-operator/pkg/config"
	"github.com/netbox-community/netbox-operator/pkg/netbox/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	asnTestRangeName = "e2e-test-asn-range"
	asnTestRangeId   = int32(77)
	asnTestRirId     = int32(9)
	asnTestPageSize  = 250
)

// asnStore is an in-memory stand-in for the NetBox ASN endpoints. Using a stateful fake
// instead of strictly ordered expectations keeps the test robust against reconcile
// retries and lets us assert on what actually ended up "in NetBox".
type asnStore struct {
	mu sync.Mutex

	asns   map[int32]*v4client.ASN
	nextId int32

	rangeStart int64
	rangeEnd   int64

	// allocated counts the ASNs handed out by the available-asns endpoint.
	allocated int
	// rangeMissing makes the ASN Range lookup return an empty result.
	rangeMissing bool
}

func newAsnStore(start, end int64) *asnStore {
	return &asnStore{
		asns:       map[int32]*v4client.ASN{},
		nextId:     100,
		rangeStart: start,
		rangeEnd:   end,
	}
}

func (s *asnStore) put(asn *v4client.ASN) {
	now := time.Now().UTC()
	asn.LastUpdated = *v4client.NewNullableTime(&now)
	s.asns[asn.Id] = asn
}

// seed inserts an ASN as if it had been created by a previous run of the operator.
func (s *asnStore) seed(value int64, customFields map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextId++
	asn := &v4client.ASN{
		Id:           s.nextId,
		Asn:          value,
		CustomFields: customFields,
		Rir:          *v4client.NewNullableBriefRIR(&v4client.BriefRIR{Id: asnTestRirId}),
	}
	s.put(asn)
}

func (s *asnStore) list() []v4client.ASN {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]v4client.ASN, 0, len(s.asns))
	for _, a := range s.asns {
		out = append(out, *a)
	}
	return out
}

func (s *asnStore) get(value int64) *v4client.ASN {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.asns {
		if a.Asn == value {
			cp := *a
			return &cp
		}
	}
	return nil
}

func okResponse(code int) *http.Response {
	return &http.Response{StatusCode: code, Body: http.NoBody}
}

// installAsnMocks wires the shared IpamAPI mock to the given store. All expectations are
// AnyTimes so that repeated reconciles do not exhaust them.
func installAsnMocks(store *asnStore) {
	mockIpamAPI.EXPECT().IpamAsnsList(gomock.Any()).
		DoAndReturn(func(_ context.Context) interfaces.IpamAsnsListRequest {
			req := mock_interfaces.NewMockIpamAsnsListRequest(mockCtrl)
			var filter []int32
			var offset int32
			req.EXPECT().Limit(int32(asnTestPageSize)).Return(req).AnyTimes()
			req.EXPECT().Offset(gomock.Any()).DoAndReturn(func(o int32) interfaces.IpamAsnsListRequest {
				offset = o
				return req
			}).AnyTimes()
			req.EXPECT().Asn(gomock.Any()).DoAndReturn(func(f []int32) interfaces.IpamAsnsListRequest {
				filter = f
				return req
			}).AnyTimes()
			req.EXPECT().Execute().DoAndReturn(func() (*v4client.PaginatedASNList, *http.Response, error) {
				all := store.list()
				matched := make([]v4client.ASN, 0, len(all))
				for _, a := range all {
					if len(filter) == 0 || int64(filter[0]) == a.Asn {
						matched = append(matched, a)
					}
				}
				if int(offset) > len(matched) {
					offset = int32(len(matched))
				}
				return &v4client.PaginatedASNList{
					Count:   int32(len(matched)),
					Results: matched[offset:],
				}, okResponse(http.StatusOK), nil
			}).AnyTimes()
			return req
		}).AnyTimes()

	mockIpamAPI.EXPECT().IpamAsnsRetrieve(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id int32) interfaces.IpamAsnsRetrieveRequest {
			req := mock_interfaces.NewMockIpamAsnsRetrieveRequest(mockCtrl)
			req.EXPECT().Execute().DoAndReturn(func() (*v4client.ASN, *http.Response, error) {
				store.mu.Lock()
				defer store.mu.Unlock()
				asn, ok := store.asns[id]
				if !ok {
					return nil, okResponse(http.StatusNotFound), nil
				}
				cp := *asn
				return &cp, okResponse(http.StatusOK), nil
			}).AnyTimes()
			return req
		}).AnyTimes()

	mockIpamAPI.EXPECT().IpamAsnsUpdate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id int32) interfaces.IpamAsnsUpdateRequest {
			req := mock_interfaces.NewMockIpamAsnsUpdateRequest(mockCtrl)
			var body v4client.ASNRequest
			req.EXPECT().ASNRequest(gomock.Any()).DoAndReturn(func(r v4client.ASNRequest) interfaces.IpamAsnsUpdateRequest {
				body = r
				return req
			}).AnyTimes()
			req.EXPECT().Execute().DoAndReturn(func() (*v4client.ASN, *http.Response, error) {
				store.mu.Lock()
				defer store.mu.Unlock()
				asn, ok := store.asns[id]
				if !ok {
					return nil, okResponse(http.StatusNotFound), nil
				}
				asn.Description = body.Description
				asn.Comments = body.Comments
				asn.CustomFields = body.CustomFields
				if body.Rir.IsSet() && body.Rir.Get() != nil && body.Rir.Get().Int32 != nil {
					asn.Rir = *v4client.NewNullableBriefRIR(&v4client.BriefRIR{Id: *body.Rir.Get().Int32})
				} else {
					asn.Rir = *v4client.NewNullableBriefRIR(nil)
				}
				store.put(asn)
				cp := *asn
				return &cp, okResponse(http.StatusOK), nil
			}).AnyTimes()
			return req
		}).AnyTimes()

	mockIpamAPI.EXPECT().IpamAsnsDestroy(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id int32) interfaces.IpamAsnsDestroyRequest {
			req := mock_interfaces.NewMockIpamAsnsDestroyRequest(mockCtrl)
			req.EXPECT().Execute().DoAndReturn(func() (*http.Response, error) {
				store.mu.Lock()
				defer store.mu.Unlock()
				delete(store.asns, id)
				return okResponse(http.StatusNoContent), nil
			}).AnyTimes()
			return req
		}).AnyTimes()

	mockIpamAPI.EXPECT().IpamAsnRangesList(gomock.Any()).
		DoAndReturn(func(_ context.Context) interfaces.IpamAsnRangesListRequest {
			req := mock_interfaces.NewMockIpamAsnRangesListRequest(mockCtrl)
			req.EXPECT().Limit(gomock.Any()).Return(req).AnyTimes()
			req.EXPECT().Offset(gomock.Any()).Return(req).AnyTimes()
			req.EXPECT().Name(gomock.Any()).Return(req).AnyTimes()
			req.EXPECT().Execute().DoAndReturn(func() (*v4client.PaginatedASNRangeList, *http.Response, error) {
				if store.rangeMissing {
					return &v4client.PaginatedASNRangeList{}, okResponse(http.StatusOK), nil
				}
				return &v4client.PaginatedASNRangeList{
					Count: 1,
					Results: []v4client.ASNRange{{
						Id:    asnTestRangeId,
						Name:  asnTestRangeName,
						Start: store.rangeStart,
						End:   store.rangeEnd,
						Rir:   v4client.BriefRIR{Id: asnTestRirId},
					}},
				}, okResponse(http.StatusOK), nil
			}).AnyTimes()
			return req
		}).AnyTimes()

	mockIpamAPI.EXPECT().IpamAsnRangesAvailableAsnsCreate(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int32) interfaces.IpamAsnRangesAvailableAsnsCreateRequest {
			req := mock_interfaces.NewMockIpamAsnRangesAvailableAsnsCreateRequest(mockCtrl)
			var body []v4client.ASNRequest
			req.EXPECT().ASNRequest(gomock.Any()).
				DoAndReturn(func(r []v4client.ASNRequest) interfaces.IpamAsnRangesAvailableAsnsCreateRequest {
					body = r
					return req
				}).AnyTimes()
			req.EXPECT().Execute().DoAndReturn(func() ([]v4client.ASN, *http.Response, error) {
				store.mu.Lock()
				defer store.mu.Unlock()

				next := store.rangeStart + int64(store.allocated)
				if next > store.rangeEnd {
					return nil, okResponse(http.StatusConflict), nil
				}
				store.allocated++
				store.nextId++

				asn := &v4client.ASN{
					Id:           store.nextId,
					Asn:          next,
					CustomFields: body[0].CustomFields,
					Description:  body[0].Description,
					Rir:          *v4client.NewNullableBriefRIR(&v4client.BriefRIR{Id: asnTestRirId}),
				}
				store.put(asn)
				return []v4client.ASN{*asn}, okResponse(http.StatusCreated), nil
			}).AnyTimes()
			return req
		}).AnyTimes()
}

var _ = Describe("AsnClaim Controller", Ordered, func() {

	const timeout = time.Second * 10
	const interval = time.Millisecond * 250

	hashKey := config.GetOperatorConfig().NetboxRestorationHashFieldName

	It("should claim an ASN and write the restoration hash at allocation time", func() {
		store := newAsnStore(64512, 64514)
		installAsnMocks(store)

		claim := &netboxv1.AsnClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "asnclaim-allocate", Namespace: "default"},
			Spec: netboxv1.AsnClaimSpec{
				ParentAsnRange: asnTestRangeName,
				Description:    "some description",
			},
		}
		expectedHash := generateAsnRestorationHash(claim)

		Expect(k8sClient.Create(ctx, claim)).To(Succeed())

		createdAsn := &netboxv1.Asn{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, createdAsn)
		}, timeout, interval).Should(Succeed())

		Expect(createdAsn.Spec.Asn).To(Equal(int64(64512)))
		Expect(createdAsn.Spec.CustomFields).To(HaveKeyWithValue(hashKey, expectedHash))
		Expect(createdAsn.OwnerReferences).To(HaveLen(1))
		Expect(createdAsn.OwnerReferences[0].Name).To(Equal(claim.Name))

		// The ASN must already carry the restoration hash from the available-asns call,
		// so that a crash before the Asn CR is reconciled cannot orphan it.
		Expect(store.get(64512)).NotTo(BeNil())
		Expect(store.get(64512).CustomFields).To(HaveKeyWithValue(hashKey, expectedHash))

		Eventually(func() bool {
			updated := &netboxv1.AsnClaim{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, updated); err != nil {
				return false
			}
			return apismeta.IsStatusConditionTrue(updated.Status.Conditions, netboxv1.ConditionAsnAssignedTrue.Type)
		}, timeout, interval).Should(BeTrue())

		// The Asn controller must reconcile the freshly allocated ASN without tripping the
		// restoration hash check and must keep the RIR intact.
		Eventually(func() bool {
			asn := &netboxv1.Asn{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, asn); err != nil {
				return false
			}
			return apismeta.IsStatusConditionTrue(asn.Status.Conditions, netboxv1.ConditionAsnReadyTrue.Type)
		}, timeout, interval).Should(BeTrue())

		Expect(store.get(64512).Rir.Get()).NotTo(BeNil())
		Expect(store.get(64512).Rir.Get().Id).To(Equal(asnTestRirId))

		// Note: the Asn CR is removed via ownerReference garbage collection, which envtest
		// does not run. We therefore delete it explicitly here, the cascading deletion is
		// covered by the chainsaw e2e tests.
		By("deleting the claim")
		Expect(k8sClient.Delete(ctx, claim)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, &netboxv1.AsnClaim{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		Expect(k8sClient.Delete(ctx, createdAsn)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, &netboxv1.Asn{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	It("should restore an ASN that already carries the restoration hash", func() {
		store := newAsnStore(64600, 64602)
		installAsnMocks(store)

		claim := &netboxv1.AsnClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "asnclaim-restore", Namespace: "default"},
			Spec: netboxv1.AsnClaimSpec{
				ParentAsnRange: asnTestRangeName,
				Description:    "some description",
			},
		}
		hash := generateAsnRestorationHash(claim)
		store.seed(64699, map[string]interface{}{hashKey: hash})

		Expect(k8sClient.Create(ctx, claim)).To(Succeed())

		createdAsn := &netboxv1.Asn{}
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, createdAsn)
		}, timeout, interval).Should(Succeed())

		// The already reserved ASN has to be reused instead of allocating a new one.
		Expect(createdAsn.Spec.Asn).To(Equal(int64(64699)))
		Expect(store.allocated).To(Equal(0))

		Expect(k8sClient.Delete(ctx, claim)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, &netboxv1.AsnClaim{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())

		Expect(k8sClient.Delete(ctx, createdAsn)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, &netboxv1.Asn{})
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	It("should report the claim as not ready when the ASN range is exhausted", func() {
		store := newAsnStore(64700, 64700)
		store.allocated = 1 // range fully used
		installAsnMocks(store)

		claim := &netboxv1.AsnClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "asnclaim-exhausted", Namespace: "default"},
			Spec: netboxv1.AsnClaimSpec{
				ParentAsnRange: asnTestRangeName,
				Description:    "some description",
			},
		}
		Expect(k8sClient.Create(ctx, claim)).To(Succeed())

		Eventually(func() bool {
			updated := &netboxv1.AsnClaim{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, updated); err != nil {
				return false
			}
			return apismeta.IsStatusConditionFalse(updated.Status.Conditions, netboxv1.ConditionAsnAssignedTrue.Type)
		}, timeout, interval).Should(BeTrue())

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: claim.Name, Namespace: claim.Namespace}, &netboxv1.Asn{})).
			To(MatchError(apierrors.IsNotFound, "not found"))

		Expect(k8sClient.Delete(ctx, claim)).To(Succeed())
	})
})
