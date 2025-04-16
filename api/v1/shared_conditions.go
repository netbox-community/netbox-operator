package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

var ConditionReadyFalseNewResource = metav1.Condition{
	Type:    "Ready",
	Status:  "False",
	Reason:  "NewResource",
	Message: "Pending Reconciliation",
}
