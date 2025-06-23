#!/bin/bash
set -e -o pipefail

# Deploy NetBox (with its PostgreSQL operator and demo data) into either:
#  • a local kind cluster (preloading images)
#  • a virtual cluster using vcluster: https://github.com/loft-sh/vcluster ( used for testing pipeline, loading of images not needed )

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Allow override via environment variable, otherwise fallback to default
NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz"

if [[ $# -lt 3 || $# -gt 4 ]]; then
    echo "Usage: $0 <CLUSTER> <VERSION> <NAMESPACE> [--vcluster]"
    exit 1
fi

CLUSTER=$1
VERSION=$2
# The specified namespace will be used for both the NetBox deployment and the vCluster creation
NAMESPACE=$3

# Force IPv4-only config for environments lacking IPv6
FORCE_NETBOX_NGINX_IPV4="${FORCE_NETBOX_NGINX_IPV4:-false}"

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
  NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz"

  # patch load-data.sh
  sed 's/netbox-demo-v4.1.sql/netbox-demo-v3.7.sql/g' $SCRIPT_DIR/load-data-job/load-data.orig.sh > $SCRIPT_DIR/load-data-job/load-data.sh && chmod +x $SCRIPT_DIR/load-data-job/load-data.sh

  # patch dockerfile (See README at https://github.com/netbox-community/pynetbox for the supported version matrix)
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
  NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.169/netbox-5.0.0-beta.169.tgz"

  # patch load-data.sh
  sed 's/netbox-demo-v4.1.sql/netbox-demo-v4.0.sql/g' $SCRIPT_DIR/load-data-job/load-data.orig.sh > $SCRIPT_DIR/load-data-job/load-data.sh && chmod +x $SCRIPT_DIR/load-data-job/load-data.sh

elif [[ "${VERSION}" == "4.1.11" ]] ;then
  echo "Using version ${VERSION}"
  # need to align with netbox-chart otherwise the creation of the cluster will hang
  declare -a Remote_Images=( \
  "busybox:1.37.0" \
  "docker.io/bitnami/redis:7.4.1-debian-12-r2" \
  "ghcr.io/netbox-community/netbox:v4.1.11" \
  "ghcr.io/zalando/postgres-operator:v1.12.2" \
  "ghcr.io/zalando/spilo-16:3.2-p3" \
  )

  # create load-data.sh
  cp $SCRIPT_DIR/load-data-job/load-data.orig.sh $SCRIPT_DIR/load-data-job/load-data.sh

else
  echo "Unknown version ${VERSION}"
  exit 1
fi

if $IS_VCLUSTER; then
  echo "[Running in vCluster mode] skipping docker pull and kind load for remote images."
else
  echo "[Running in Kind mode] pulling and loading remote images into kind cluster..."
  for img in "${Remote_Images[@]}"; do
    docker pull "$img"
    kind load docker-image "$img" --name "${CLUSTER}"
  done
fi

# build image for loading local data via NetBox API
cd "$SCRIPT_DIR/load-data-job"

# Assign IMAGE_REGISTRY from env if set, else empty
POSTGRES_IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"

# Build optional set flag if registry is not defined
REGISTRY_ARG=""
if [ -n "$POSTGRES_IMAGE_REGISTRY" ]; then
  REGISTRY_ARG="--set image.registry=$POSTGRES_IMAGE_REGISTRY"
fi

# Install Postgres Operator
# Allow override via environment variable, otherwise fallback to default
POSTGRES_OPERATOR_HELM_CHART="${POSTGRES_OPERATOR_HELM_REPO:-https://opensource.zalando.com/postgres-operator/charts/postgres-operator}/postgres-operator-1.12.2.tgz"
${HELM} upgrade --install postgres-operator "$POSTGRES_OPERATOR_HELM_CHART" \
    --namespace="${NAMESPACE}" \
    --create-namespace \
    --set podPriorityClassName.create=false \
    --set podServiceAccount.name="postgres-pod-${NAMESPACE}" \
    --set serviceAccount.name="postgres-operator-${NAMESPACE}" \
    $REGISTRY_ARG

# Deploy the database
export SPILO_IMAGE="${IMAGE_REGISTRY:-ghcr.io}/zalando/spilo-16:3.2-p3"
echo "spilo image is $SPILO_IMAGE"
envsubst < "$SCRIPT_DIR/netbox-db/netbox-db-patch.tmpl.yaml" > "$SCRIPT_DIR/netbox-db/netbox-db-patch.yaml"
${KUBECTL} apply -n "$NAMESPACE" -k "$SCRIPT_DIR/netbox-db"
rm "$SCRIPT_DIR/netbox-db/netbox-db-patch.yaml"

echo "loading demo-data into NetBox…"
kubectl create configmap netbox-demo-data-load-job-scripts \
  --from-file="$SCRIPT_DIR/load-data-job" \
  --dry-run=client -o yaml \
| ${KUBECTL} apply -n "${NAMESPACE}" -f -

# Set the image of the kustomization.yaml to the one specified (from env or default)
SPILO_IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io}"
SPILO_IMAGE="${SPILO_IMAGE_REGISTRY}/zalando/spilo-16:3.2-p3"

JOB_DIR="$SCRIPT_DIR/job"
cd "$JOB_DIR"
kustomize edit set image ghcr.io/zalando/spilo-16="$SPILO_IMAGE"

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
# reset the kustomization to default value
rm sql-env-patch.yaml
kustomize edit set image ghcr.io/zalando/spilo-16="ghcr.io/zalando/spilo-16"
cd ..

${KUBECTL} wait \
    -n "${NAMESPACE}" --for=condition=complete --timeout=600s job/netbox-demo-data-load-job

${KUBECTL} delete \
    -n "${NAMESPACE}" configmap/netbox-demo-data-load-job-scripts

# Assign IMAGE_REGISTRY from env if set, else empty
NETBOX_IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"

# Build optional set flag if registry is not defined
REGISTRY_ARG=""
if [ -n "$NETBOX_IMAGE_REGISTRY" ]; then
  REGISTRY_ARG="--set global.imageRegistry=$NETBOX_IMAGE_REGISTRY --set global.security.allowInsecureImages=true"
fi

# Install NetBox
${HELM} upgrade --install netbox ${NETBOX_HELM_CHART} \
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
  $REGISTRY_ARG

if [[ "$FORCE_NETBOX_NGINX_IPV4" == "true" ]]; then
  echo "Creating nginx-unit ConfigMap and patching deployment"

  ${KUBECTL} apply -f "$SCRIPT_DIR/nginx-unit-config.yaml" -n "$NAMESPACE"

  ${KUBECTL} patch deployment netbox -n "$NAMESPACE" --type=json -p='[
    {
      "op": "add",
      "path": "/spec/template/spec/volumes/-",
      "value": {
        "name": "unit-config",
        "configMap": {
          "name": "nginx-unit-config"
        }
      }
    },
    {
      "op": "add",
      "path": "/spec/template/spec/containers/0/volumeMounts/-",
      "value": {
        "mountPath": "/etc/unit/nginx-unit.json",
        "subPath": "nginx-unit.json",
        "name": "unit-config"
      }
    }
  ]'

fi

${KUBECTL} rollout status --namespace="${NAMESPACE}" deployment netbox

# Create ConfigMap for the Python script
TMP_CONFIGMAP_YAML="$(mktemp)"
kubectl create configmap netbox-loader-script \
  --namespace="${NAMESPACE}" \
  --from-file=main.py="$SCRIPT_DIR/load-local-data-job/main.py" \
  --dry-run=client -o yaml > "$TMP_CONFIGMAP_YAML"

${KUBECTL} apply -f "$TMP_CONFIGMAP_YAML" --namespace="${NAMESPACE}"
rm "$TMP_CONFIGMAP_YAML"

# Prepare Job YAML with optional environment variable injection
JOB_YAML="$SCRIPT_DIR/load-local-data-job/netbox-load-local-data-job.yaml"
TMP_JOB_YAML="$(mktemp)"
cp "$JOB_YAML" "$TMP_JOB_YAML"

# Define internal NetBox service endpoint (used in Kind)
NETBOX_API_URL="http://netbox.${NAMESPACE}.svc.cluster.local"

PATCHED_TMP_JOB_YAML="$(mktemp)"

# Convert YAML to JSON and inject variables if containers exist
yq -o=json "$TMP_JOB_YAML" | jq \
  --arg netboxApi "$NETBOX_API_URL" \
  --arg pypiUrl "$PYPI_REPOSITORY_URL" \
  --arg artifactoryHost "$ARTIFACTORY_TRUSTED_HOST" \
  --arg imageRegistry "${IMAGE_REGISTRY:-docker.io}" '
  .spec.template.spec.containers[0].env //= [] |
  .spec.template.spec.containers[0].image = $imageRegistry+"/python:3.12-slim" |
  .spec.template.spec.containers[0].env +=
    [{"name": "NETBOX_API", "value": $netboxApi}]
    + (
        if $pypiUrl != "" and $artifactoryHost != "" then
          [
            {"name": "PYPI_REPOSITORY_URL", "value": $pypiUrl},
            {"name": "ARTIFACTORY_TRUSTED_HOST", "value": $artifactoryHost}
          ]
        else [] end
      )
' | yq -P > "$PATCHED_TMP_JOB_YAML"

mv "$PATCHED_TMP_JOB_YAML" "$TMP_JOB_YAML"

# Delete previous job if it exists
${KUBECTL} delete job netbox-load-local-data --namespace="${NAMESPACE}" --ignore-not-found

# Apply patched job
${KUBECTL} apply -n "${NAMESPACE}" -f "$TMP_JOB_YAML"
rm "$TMP_JOB_YAML"

# Wait for job to complete
${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=condition=complete job/netbox-load-local-data

# Load local data
${KUBECTL} delete job netbox-load-local-data --namespace="${NAMESPACE}"
${KUBECTL} delete configmap netbox-loader-script --namespace="${NAMESPACE}"

# clean up
rm $SCRIPT_DIR/load-data-job/load-data.sh
