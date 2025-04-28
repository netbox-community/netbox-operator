#!/bin/bash
set -e -u -o pipefail

NETBOX_HELM_CHART="https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz" # default value

if [[ $# -lt 3 || $# -gt 4 ]]; then
    echo "Usage: $0 <CLUSTER> <VERSION> <NAMESPACE> [--vcluster]"
    exit 1
fi

CLUSTER=$1
VERSION=$2
NAMESPACE=$3
VCLUSTER_MODE=${4:-}

# Choose kubectl and helm commands depending if we run on vCluster
if [[ "$VCLUSTER_MODE" == "--vcluster" ]]; then
    KUBECTL="vcluster connect ${CLUSTER} -n ${NAMESPACE} -- kubectl"
    HELM="vcluster connect ${CLUSTER} -n ${NAMESPACE} -- helm"
else
    KUBECTL="kubectl"
    HELM="helm"
fi

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

if [[ "$VCLUSTER_MODE" == "--vcluster" ]]; then
  echo "[Running in vCluster mode] skipping docker pull and kind load for remote images."
  sleep 15
else
  echo "[Running in Kind mode] pulling and loading remote images into kind cluster..."
  for img in "${Remote_Images[@]}"; do
    docker pull "$img"
    kind load docker-image "$img" --name "${CLUSTER}"
  done
fi

# build image for loading local data via NetBox API
cd "$(dirname "$0")/load-data-job"
docker build -t netbox-load-local-data:1.0 --load --no-cache --progress=plain -f ./dockerfile .
cd -

# Load local images into Kind only if not vCluster
if [[ "$VCLUSTER_MODE" != "--vcluster" ]]; then
  echo "Loading local images into kind cluster..."
  declare -a Local_Images=( \
  "netbox-load-local-data:1.0" \
  )
  for img in "${Local_Images[@]}"; do
    kind load docker-image "$img" --name "${CLUSTER}"
  done
else
  echo "Skipping local image loading into Kind (vCluster mode)."
fi

# Install Postgres Operator
${HELM} upgrade --install postgres-operator \
  --namespace="${NAMESPACE}" \
  --create-namespace \
  --set podPriorityClassName.create=false \
  --set podServiceAccount.name="postgres-pod-${NAMESPACE}" \
  --set serviceAccount.name="postgres-operator-${NAMESPACE}" \
  https://opensource.zalando.com/postgres-operator/charts/postgres-operator/postgres-operator-1.12.2.tgz

# Deploy the database
${KUBECTL} apply --namespace="${NAMESPACE}" -f "$(dirname "$0")/netbox-db.yaml"
${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=jsonpath='{.status.PostgresClusterStatus}'=Running postgresql/netbox-db

# Load demo data
if [[ -d "$(dirname "$0")/load-data-job" ]]; then
  echo "load-data-job directory found, creating ConfigMap."

  echo "Generating ConfigMap YAML:"
  ${KUBECTL} create configmap --namespace="${NAMESPACE}" netbox-demo-data-load-job-scripts --from-file="/tmp/netbox-operator/kind/load-data-job" -o yaml --dry-run=client | ${KUBECTL} apply -f -

  echo "Applying ConfigMap to Kubernetes:"
  ${KUBECTL} apply -f /tmp/netbox-configmap.yaml

  echo "Deploying load-data-job.yaml:"
  ${KUBECTL} apply --namespace="${NAMESPACE}" -f "$(dirname "$0")/load-data-job.yaml"

  echo "Waiting for load-data-job to complete:"
  ${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=condition=complete job/netbox-demo-data-load-job

  echo "Cleaning up ConfigMap:"
  ${KUBECTL} delete configmap --namespace="${NAMESPACE}" netbox-demo-data-load-job-scripts
else
  echo " Skipping load-data-job because directory does not exist."
fi

# Install NetBox
${HELM} upgrade --install netbox \
  --namespace="${NAMESPACE}" \
  --create-namespace \
  --set postgresql.enabled="false" \
  --set externalDatabase.host="netbox-db.${NAMESPACE}.svc.cluster.local" \
  --set externalDatabase.existingSecretName="netbox.netbox-db.credentials.postgresql.acid.zalan.do" \
  --set externalDatabase.existingSecretKey="password" \
  --set redis.auth.password="password" \
  --set resources.requests.cpu="500m" \
  --set resources.requests.memory="512Mi" \
  --set resources.limits.cpu="2000m" \
  --set resources.limits.memory="2Gi" \
  ${NETBOX_HELM_CHART}

${KUBECTL} rollout status --namespace="${NAMESPACE}" deployment netbox

# Load local data
${KUBECTL} delete job netbox-load-local-data --namespace="${NAMESPACE}" --ignore-not-found
${KUBECTL} create job netbox-load-local-data --namespace="${NAMESPACE}" --image=netbox-load-local-data:1.0
${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=condition=complete job/netbox-load-local-data

# clean up
rm $(dirname "$0")/load-data-job/load-data.sh
rm $(dirname "$0")/load-data-job/dockerfile
