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

# image for loading local data via NetBox API
cd ./kind/load-data-job && docker build -t netbox-load-local-data:1.0 -f ./dockerfile . && cd -

# need to align with netbox-chart otherwise the creation of the cluster will hang
declare -a Images=( \
"gcr.io/kubebuilder/kube-rbac-proxy:v0.14.1" \
"busybox:1.37.0" \
"docker.io/bitnami/redis:7.4.1-debian-12-r2" \
"ghcr.io/netbox-community/netbox:v4.1.7" \
"ghcr.io/zalando/postgres-operator:v1.12.2" \
"ghcr.io/zalando/spilo-16:3.2-p3" \
"netbox-load-local-data:1.0" \
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

helm upgrade --install --namespace="${NAMESPACE}" netbox \
  --set postgresql.enabled="false" \
  --set externalDatabase.host="netbox-db.${NAMESPACE}.svc.cluster.local" \
  --set externalDatabase.existingSecretName="netbox.netbox-db.credentials.postgresql.acid.zalan.do" \
  --set externalDatabase.existingSecretKey="password" \
  --set redis.auth.password="password" \
  https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.163/netbox-5.0.0-beta.163.tgz

kubectl rollout status --namespace="${NAMESPACE}" deployment netbox

# load local data
kubectl create job netbox-load-local-data --image=netbox-load-local-data:1.0
kubectl wait --namespace="${NAMESPACE}"  --timeout=600s --for=condition=complete job/netbox-load-local-data
docker rmi netbox-load-local-data:1.0
