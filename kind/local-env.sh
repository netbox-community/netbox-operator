#!/bin/bash
set -e -u -o pipefail

NAMESPACE=""
VERSION="4.1.7" # default value (latest)
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--namespace)
      NAMESPACE="$2"
      shift # past argument
      shift # past value
      ;;
    -v|--version)
      VERSION="$2"
      shift # past argument
      shift # past value
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

echo "=======Parsed arguments======="
echo "Namespace   = ${NAMESPACE}"
echo "Version     = ${VERSION}"
echo "=============================="

# aurgment check / init
if [ -z "$NAMESPACE" ]; then
    echo "Using default namespace"
    NAMESPACE="default"
else
    echo "Using namespace: $NAMESPACE"
fi

if [[ "${VERSION}" == "3.7.8" ]] ;then
  echo "Using version ${VERSION}"
elif [[ "${VERSION}" == "4.0.11" ]] ;then
  echo "Using version ${VERSION}"
elif [[ "${VERSION}" == "4.1.7" ]] ;then
  echo "Using version ${VERSION}"
else
  echo "Unknown version ${VERSION}"
  exit 1
fi

# create a kind cluster
kind create cluster || echo "cluster already exists, continuing..."

# Add a delay here, in case we run into "Namespace default does not exist."
sleep 1

# deal with namespace
if ! kubectl get namespaces | grep -q "^${NAMESPACE} "; then
    echo "Namespace ${NAMESPACE} does not exist."
    exit 1
fi

# need to align with netbox-chart otherwise the creation of the cluster will hang
declare -a Images=( \
"gcr.io/kubebuilder/kube-rbac-proxy:v0.14.1" \
"busybox:1.37.0" \
"docker.io/bitnami/redis:7.4.1-debian-12-r2" \
"ghcr.io/netbox-community/netbox:v4.1.7" \
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

helm upgrade --install --namespace="${NAMESPACE}" netbox \
  --set postgresql.enabled="false" \
  --set externalDatabase.host="netbox-db.${NAMESPACE}.svc.cluster.local" \
  --set externalDatabase.existingSecretName="netbox.netbox-db.credentials.postgresql.acid.zalan.do" \
  --set externalDatabase.existingSecretKey="password" \
  --set redis.auth.password="password" \
  https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.163/netbox-5.0.0-beta.163.tgz

kubectl rollout status --namespace="${NAMESPACE}" deployment netbox
