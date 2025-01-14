#!/bin/bash
set -e -u -o pipefail

NAMESPACE=""
VERSION="4.1.8" # default value
NETBOX_HELM_CHART="https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz" # default value
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

# create a kind cluster
kind create cluster || echo "cluster already exists, continuing..."
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=1s namespace/${NAMESPACE}

# load remote images
if [[ "${VERSION}" == "3.7.8" ]] ;then
  echo "Using version ${VERSION}"
  # need to align with netbox-chart otherwise the creation of the cluster will hang
  declare -a Remote_Images=( \
  "busybox:1.36.1" \
  "docker.io/bitnami/redis:7.2.4-debian-12-r9" \
  "docker.io/netboxcommunity/netbox:v3.7.8" \
  "ghcr.io/zalando/postgres-operator:v1.12.2" \
  "ghcr.io/zalando/spilo-16:3.2-p3" \
  )
  NETBOX_HELM_CHART="https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta5/netbox-5.0.0-beta5.tgz"

  # patch load-data.sh
  sed 's/netbox-demo-v4.1.sql/netbox-demo-v3.7.sql/g' $(dirname "$0")/load-data-job/load-data.orig.sh > $(dirname "$0")/load-data-job/load-data.sh && chmod +x $(dirname "$0")/load-data-job/load-data.sh

  # patch dockerfile (See README at https://github.com/netbox-community/pynetbox for the supported version matrix)
  sed 's/RUN pip install -Iv pynetbox==7.4.1/RUN pip install -Iv pynetbox==7.3.4/g' $(dirname "$0")/load-data-job/dockerfile.orig > $(dirname "$0")/load-data-job/dockerfile
elif [[ "${VERSION}" == "4.0.11" ]] ;then
  echo "Using version ${VERSION}"
  # need to align with netbox-chart otherwise the creation of the cluster will hang
  declare -a Remote_Images=( \
  "busybox:1.36.1" \
  "docker.io/bitnami/redis:7.4.0-debian-12-r2" \
  "ghcr.io/netbox-community/netbox:v4.0.11" \
  "ghcr.io/zalando/postgres-operator:v1.12.2" \
  "ghcr.io/zalando/spilo-16:3.2-p3" \
  )
  NETBOX_HELM_CHART="https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.84/netbox-5.0.0-beta.84.tgz"
  
  # patch load-data.sh
  sed 's/netbox-demo-v4.1.sql/netbox-demo-v4.0.sql/g' $(dirname "$0")/load-data-job/load-data.orig.sh > $(dirname "$0")/load-data-job/load-data.sh && chmod +x $(dirname "$0")/load-data-job/load-data.sh
  
  cp $(dirname "$0")/load-data-job/dockerfile.orig $(dirname "$0")/load-data-job/dockerfile
elif [[ "${VERSION}" == "4.1.8" ]] ;then
  echo "Using version ${VERSION}"
  # need to align with netbox-chart otherwise the creation of the cluster will hang
  declare -a Remote_Images=( \
  "busybox:1.37.0" \
  "docker.io/bitnami/redis:7.4.1-debian-12-r2" \
  "ghcr.io/netbox-community/netbox:v4.1.8" \
  "ghcr.io/zalando/postgres-operator:v1.12.2" \
  "ghcr.io/zalando/spilo-16:3.2-p3" \
  )

  # create load-data.sh
  cp $(dirname "$0")/load-data-job/load-data.orig.sh $(dirname "$0")/load-data-job/load-data.sh

  cp $(dirname "$0")/load-data-job/dockerfile.orig $(dirname "$0")/load-data-job/dockerfile
else
  echo "Unknown version ${VERSION}"
  exit 1
fi

for img in "${Remote_Images[@]}"; do
  docker pull "$img"
  kind load docker-image "$img"
done

# build image for loading local data via NetBox API
cd ./kind/load-data-job && docker build -t netbox-load-local-data:1.0 --load --no-cache --progress=plain -f ./dockerfile . && cd -

# load local images
declare -a Local_Images=( \
"netbox-load-local-data:1.0" \
)
for img in "${Local_Images[@]}"; do
  kind load docker-image "$img"
done

# install helm charts
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
  ${NETBOX_HELM_CHART}

kubectl rollout status --namespace="${NAMESPACE}" deployment netbox

# load local data
kubectl create job netbox-load-local-data --image=netbox-load-local-data:1.0
kubectl wait --namespace="${NAMESPACE}"  --timeout=600s --for=condition=complete job/netbox-load-local-data
docker rmi netbox-load-local-data:1.0

# clean up
rm $(dirname "$0")/load-data-job/load-data.sh
rm $(dirname "$0")/load-data-job/dockerfile
