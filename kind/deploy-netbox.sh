#!/bin/bash
set -e -u -o pipefail

# Deploy NetBox (with its PostgreSQL operator and demo data) into either:
#  • a local kind cluster (preloading images)
#  • a virtual cluster using vcluster: https://github.com/loft-sh/vcluster ( used for testing pipeline, loading of images not needed )

# Allow override via environment variable, otherwise fallback to default
NETBOX_HELM_CHART="${NETBOX_HELM_CHART:-https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz}"

if [[ $# -lt 3 || $# -gt 4 ]]; then
    echo "Usage: $0 <CLUSTER> <VERSION> <NAMESPACE> [--vcluster]"
    exit 1
fi

CLUSTER=$1
VERSION=$2
NAMESPACE=$3

# Treat the optional fourth argument "--vcluster" as a boolean flag
IS_VCLUSTER=false
if [[ "${4:-}" == "--vcluster" ]]; then
    IS_VCLUSTER=true
fi

# Choose kubectl and helm commands depending if we run on vCluster
if $IS_VCLUSTER; then
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
  # Allow override via environment variable, otherwise fallback to default
  NETBOX_HELM_CHART="${NETBOX_HELM_CHART:-https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta5/netbox-5.0.0-beta5.tgz}"

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
  # Allow override via environment variable, otherwise fallback to default
  NETBOX_HELM_CHART="${NETBOX_HELM_CHART:-https://github.com/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.84/netbox-5.0.0-beta.84.tgz}"

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

if $IS_VCLUSTER; then
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
docker build -t netbox-load-local-data:1.0 \
  --load --no-cache --progress=plain \
  --build-arg PYTHON_BASE_IMAGE="${PYTHON_BASE_IMAGE:-python:3.12}" \
  --build-arg ARTIFACTORY_PYPI_URL="${ARTIFACTORY_PYPI_URL:-}" \
  --build-arg ARTIFACTORY_TRUSTED_HOST="${ARTIFACTORY_TRUSTED_HOST:-}" \
  -f ./dockerfile .
cd -

# Load local images into Kind only if not vCluster
if ! $IS_VCLUSTER; then
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
# Allow override via environment variable, otherwise fallback to default
POSTGRESS_OPERATOR_HELM_CHART="${POSTGRESS_OPERATOR_HELM_CHART:-https://opensource.zalando.com/postgres-operator/charts/postgres-operator/postgres-operator-1.12.2.tgz}"
${HELM} upgrade --install postgres-operator \
  --namespace="${NAMESPACE}" \
  --create-namespace \
  --set podPriorityClassName.create=false \
  --set podServiceAccount.name="postgres-pod-${NAMESPACE}" \
  --set serviceAccount.name="postgres-operator-${NAMESPACE}" \
   "${POSTGRESS_OPERATOR_HELM_CHART}"

# Deploy the database
${KUBECTL} apply --namespace="${NAMESPACE}" -f "$(dirname "$0")/netbox-db.yaml"
${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=jsonpath='{.status.PostgresClusterStatus}'=Running postgresql/netbox-db

echo "loading demo-data into NetBox…"
# We use plain `kubectl create … --dry-run=client -o yaml` here to generate
# the ConfigMap manifest locally (no cluster connection needed), then pipe
# that YAML into `${KUBECTL} apply` so it’s applied against the selected
# target (Kind or vCluster) via our `${KUBECTL}` wrapper.
if $IS_VCLUSTER; then
  # — vCluster —
  echo "  → inside the vcluster"
  kubectl create configmap netbox-demo-data-load-job-scripts \
    --from-file="$(dirname "$0")/load-data-job" \
    --dry-run=client -o yaml \
  | ${KUBECTL} apply -n "${NAMESPACE}" -f -

else
  # — Kind —
  echo "  → on the Kind cluster (${NAMESPACE})"
  ${KUBECTL} create configmap netbox-demo-data-load-job-scripts \
    --from-file="$(dirname "$0")/load-data-job" \
    --namespace="${NAMESPACE}" \
    --dry-run=client -o yaml \
  | ${KUBECTL} apply -f -
fi

# Set the image of the kustomization.yaml to the one specified (from env or default)
JOB_DIR="$(dirname "$0")/job"
cd "$JOB_DIR"
kustomize edit set image ghcr.io/zalando/spilo-16="${SPILO_IMAGE:-ghcr.io/zalando/spilo-16:3.2-p3}"

# Create a patch file to inject NETBOX_SQL_DUMP_URL (from env or default)
NETBOX_SQL_DUMP_URL="${NETBOX_SQL_DUMP_URL:-https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.1.sql}"

# Create patch
cat > sql-env-patch.yaml <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: netbox-demo-data-load-job
spec:
  template:
    spec:
      containers:
        - name: netbox-demo-data-load
          env:
            - name: NETBOX_SQL_DUMP_URL
              value: "${NETBOX_SQL_DUMP_URL}"
EOF

# Add the patch
kustomize edit add patch --path sql-env-patch.yaml

# Apply the customized job
kustomize build . | ${KUBECTL} apply -n "${NAMESPACE}" -f -
cd ..

${KUBECTL} wait \
    -n "${NAMESPACE}" --for=condition=complete --timeout=600s job/netbox-demo-data-load-job

${KUBECTL} delete \
    -n "${NAMESPACE}" configmap/netbox-demo-data-load-job-scripts


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
