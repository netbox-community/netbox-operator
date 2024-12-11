#!/bin/bash
set -o errexit

kind create cluster || echo "cluster already exists, continuing..."

if [ -z "$1" ]; then
    echo "Using default namespace."
    NAMESPACE="default"
else
    echo "Using namespace: $1"
    NAMESPACE="$1"
fi
if ! kubectl get namespaces | grep -q "^${NAMESPACE} "; then
    echo "Namespace ${NAMESPACE} does not exist."
    exit 1
fi

declare -a Images=( \
"gcr.io/kubebuilder/kube-rbac-proxy:v0.14.1" \
"busybox:1.36.1" \
"bitnami/redis:7.2.5-debian-12-r0" \
"ghcr.io/netbox-community/netbox:v4.0.5" \
"ghcr.io/zalando/postgres-operator:v1.12.2" \
"ghcr.io/zalando/spilo-16:3.2-p3" \
)

for img in "${Images[@]}"; do
  docker pull "$img"
  kind load docker-image "$img"
done

helm upgrade --install --namespace="${NAMESPACE}" postgres-operator \
https://opensource.zalando.com/postgres-operator/charts/postgres-operator/postgres-operator-1.12.2.tgz

kubectl apply --namespace="${NAMESPACE}" -f "$(dirname "$0")/netbox-db.yaml"
kubectl wait --namespace="${NAMESPACE}"  --timeout=600s --for=jsonpath='{.status.PostgresClusterStatus}'=Running postgresql/netbox-db

kubectl create configmap --namespace="${NAMESPACE}" netbox-demo-data-load-job-scripts --from-file="$(dirname "$0")/load-data-job" -o yaml --dry-run=client | kubectl apply -f -
kubectl apply --namespace="${NAMESPACE}" -f "$(dirname "$0")/load-data-job.yaml"
kubectl wait --namespace="${NAMESPACE}"  --timeout=600s --for=condition=complete job/netbox-demo-data-load-job
kubectl delete configmap --namespace="${NAMESPACE}" netbox-demo-data-load-job-scripts
kubectl delete --namespace="${NAMESPACE}" -f "$(dirname "$0")/load-data-job.yaml"

# 7. Helm install
helm upgrade --install --namespace="${NAMESPACE}" netbox \
  --set postgresql.enabled="false" \
  --set externalDatabase.host="netbox-db.${NAMESPACE}.svc.cluster.local" \
  --set externalDatabase.existingSecretName="netbox.netbox-db.credentials.postgresql.acid.zalan.do" \
  --set externalDatabase.existingSecretKey="password" \
  --set redis.auth.password="password" \
  https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.161/netbox-5.0.0-beta.161.tgz

kubectl rollout status --namespace="${NAMESPACE}" deployment netbox

