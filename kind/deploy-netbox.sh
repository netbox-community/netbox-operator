#!/bin/bash
set -e -u -o pipefail

# Deploy NetBox (with its PostgreSQL operator and demo data) into either:
#  • a local kind cluster (preloading images)
#  • a virtual cluster using vcluster: https://github.com/loft-sh/vcluster ( used for testing pipeline, loading of images not needed )

log() {
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $*"
}

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Files generated during the run, removed on exit so a re-run always starts from a clean tree
GENERATED_FILES=(
    "$SCRIPT_DIR/netbox-db/netbox-db-patch.yaml"
    "$SCRIPT_DIR/job/kustomization.yaml"
    "$SCRIPT_DIR/job/sql-env-patch.yaml"
)
cleanup() {
    rm -f "${GENERATED_FILES[@]}"
}
trap cleanup EXIT

# Allow override via environment variable, otherwise fallback to default
NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.9/netbox-5.0.9.tgz"

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

# Set to DEBUG to also get request-level detail out of NetBox
NETBOX_LOG_LEVEL="${NETBOX_LOG_LEVEL:-INFO}"

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

# Warn early: the script only ever does `helm upgrade --install`, so a version change
# reuses the existing release, PVCs and cache secrets.
if ${KUBECTL} get deployment netbox -n "${NAMESPACE}" > /dev/null 2>&1; then
    CURRENT_IMAGE="$(${KUBECTL} get deployment netbox -n "${NAMESPACE}" \
        -o jsonpath='{.spec.template.spec.containers[0].image}')"
    CURRENT_VERSION="${CURRENT_IMAGE##*:}"
    CURRENT_VERSION="${CURRENT_VERSION#v}"
    if [[ -n "${CURRENT_VERSION}" && "${CURRENT_VERSION}" != "${VERSION}" ]]; then
        log "WARNING: NetBox ${CURRENT_VERSION} is already deployed in namespace ${NAMESPACE}."
        log "WARNING: You are changing the version in place to ${VERSION}. This is not well tested and might not work."
        log "WARNING: If it fails (e.g. a helm password error on upgrade), delete the cluster and start fresh."
    fi
fi

log "Resolving helm chart and demo data for NetBox ${VERSION}"
if [[ "${VERSION}" == "3.7.8" ]] ;then
  NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta5/netbox-5.0.0-beta5.tgz"
  NETBOX_SQL_DUMP_URL="https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v3.7.sql"

elif [[ "${VERSION}" == "4.0.11" ]] ;then
  NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-5.0.0-beta.84/netbox-5.0.0-beta.84.tgz"
  NETBOX_SQL_DUMP_URL="https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.0.sql"

elif [[ "${VERSION}" == "4.1.10" ]] ;then
  log "Using default helm chart and demo data"

elif [[ "${VERSION}" == "4.4.9" ]] ;then
  NETBOX_HELM_CHART="${NETBOX_HELM_REPO:-https://github.com}/netbox-community/netbox-chart/releases/download/netbox-7.2.26/netbox-7.2.26.tgz"
  NETBOX_SQL_DUMP_URL="https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.4.sql"

else
  echo "Unknown version ${VERSION}"
  exit 1
fi

# Assign IMAGE_REGISTRY from env if set, else empty
POSTGRES_IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"

# Build optional set flag if registry is not defined
REGISTRY_ARG=""
if [ -n "$POSTGRES_IMAGE_REGISTRY" ]; then
  REGISTRY_ARG="--set image.registry=$POSTGRES_IMAGE_REGISTRY"
fi

log "Installing the Postgres operator"
# Allow override via environment variable, otherwise fallback to default
POSTGRES_OPERATOR_HELM_CHART="${POSTGRES_OPERATOR_HELM_REPO:-https://opensource.zalando.com/postgres-operator/charts/postgres-operator}/postgres-operator-1.12.2.tgz"
${HELM} upgrade --install postgres-operator "$POSTGRES_OPERATOR_HELM_CHART" \
    --namespace="${NAMESPACE}" \
    --create-namespace \
    --set podPriorityClassName.create=false \
    --set podServiceAccount.name="postgres-pod-${NAMESPACE}" \
    --set serviceAccount.name="postgres-operator-${NAMESPACE}" \
    $REGISTRY_ARG

export SPILO_IMAGE="${IMAGE_REGISTRY:-ghcr.io}/zalando/spilo-16:3.2-p3"
log "Deploying the database with spilo image $SPILO_IMAGE"
envsubst < "$SCRIPT_DIR/netbox-db/netbox-db-patch.tmpl.yaml" > "$SCRIPT_DIR/netbox-db/netbox-db-patch.yaml"
${KUBECTL} apply -n "$NAMESPACE" -k "$SCRIPT_DIR/netbox-db"

# The operator creates the StatefulSet only some time after the CR is applied, and
# gives it an OnDelete update strategy, which rules out `kubectl rollout status`.
log "Waiting for the netbox-db statefulset to become ready"
${KUBECTL} wait -n "$NAMESPACE" --for=create statefulset/netbox-db --timeout=300s
${KUBECTL} wait -n "$NAMESPACE" --for=jsonpath='{.status.readyReplicas}'=1 \
    statefulset/netbox-db --timeout=600s

log "Creating the demo-data load job scripts ConfigMap"
kubectl create configmap netbox-demo-data-load-job-scripts \
  --from-file="$SCRIPT_DIR/load-data-job" \
  --dry-run=client -o yaml \
| ${KUBECTL} apply -n "${NAMESPACE}" -f -

# Set the image of the kustomization.yaml to the one specified (from env or default)
SPILO_IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io}"
SPILO_IMAGE="${SPILO_IMAGE_REGISTRY}/zalando/spilo-16:3.2-p3"

JOB_DIR="$SCRIPT_DIR/job"
cp "$JOB_DIR/kustomization.orig.yaml" "$JOB_DIR/kustomization.yaml"

NETBOX_SQL_DUMP_URL="${NETBOX_SQL_DUMP_URL:-https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.1.sql}"

log "Patching the load job with NETBOX_SQL_DUMP_URL=${NETBOX_SQL_DUMP_URL} and image ${SPILO_IMAGE}"
cat > "$JOB_DIR/sql-env-patch.yaml" <<EOF
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

# kustomize edit only operates on the kustomization.yaml in the current directory
(
    cd "$JOB_DIR"
    kustomize edit set image ghcr.io/zalando/spilo-16="$SPILO_IMAGE"
    kustomize edit add patch --path sql-env-patch.yaml
)

# The demo data dump is a plain pg_dump, so the load job recreates the public schema.
# No NetBox pod may hold connections to it while that happens.
NETBOX_SCALED_DOWN=false
if ${KUBECTL} get deployment netbox -n "${NAMESPACE}" > /dev/null 2>&1; then
    log "Scaling down NetBox while the database is reloaded"
    ${KUBECTL} scale deployment netbox -n "${NAMESPACE}" --replicas=0
    ${KUBECTL} delete pods -n "${NAMESPACE}" -l app.kubernetes.io/component=netbox --wait=true
    NETBOX_SCALED_DOWN=true
fi

# A completed Job cannot be updated in place, so replace it on every run
${KUBECTL} delete job netbox-demo-data-load-job -n "${NAMESPACE}" --ignore-not-found --wait=true

log "Loading demo-data into NetBox"
kustomize build "$JOB_DIR" | ${KUBECTL} apply -n "${NAMESPACE}" -f -
${KUBECTL} wait \
    -n "${NAMESPACE}" --for=condition=complete --timeout=600s job/netbox-demo-data-load-job

log "Demo-data load job completed"
${KUBECTL} delete --ignore-not-found \
    -n "${NAMESPACE}" configmap/netbox-demo-data-load-job-scripts

# Assign IMAGE_REGISTRY from env if set, else empty
NETBOX_IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"

# Build optional set flag if registry is not defined
REGISTRY_ARG=""
if [ -n "$NETBOX_IMAGE_REGISTRY" ]; then
  REGISTRY_ARG="--set global.imageRegistry=$NETBOX_IMAGE_REGISTRY --set global.security.allowInsecureImages=true"
fi

# Rendered verbatim into NetBox's configuration.py as the Django LOGGING dict.
# Without it NetBox logs nothing but NGINX Unit access lines, so a crashing pod
# leaves no trace of why it crashed. django.request is what reports HTTP 5xx.
NETBOX_LOGGING_JSON="{\"version\":1,\"disable_existing_loggers\":false,\"formatters\":{\"verbose\":{\"format\":\"[%(asctime)s] %(levelname)s %(name)s %(message)s\"}},\"handlers\":{\"console\":{\"class\":\"logging.StreamHandler\",\"formatter\":\"verbose\"}},\"root\":{\"handlers\":[\"console\"],\"level\":\"${NETBOX_LOG_LEVEL}\"},\"loggers\":{\"django\":{\"handlers\":[\"console\"],\"level\":\"${NETBOX_LOG_LEVEL}\",\"propagate\":false},\"django.request\":{\"handlers\":[\"console\"],\"level\":\"DEBUG\",\"propagate\":false},\"netbox\":{\"handlers\":[\"console\"],\"level\":\"${NETBOX_LOG_LEVEL}\",\"propagate\":false}}}"

# The chart defaults SIGTERM a healthy but slow pod on kind, hence we fix here.
PROBE_ARGS=(
  --set livenessProbe.timeoutSeconds=5
  --set livenessProbe.periodSeconds=15
  --set livenessProbe.failureThreshold=10
  --set readinessProbe.timeoutSeconds=5
  --set startupProbe.timeoutSeconds=5
  --set startupProbe.failureThreshold=60
)

log "Installing NetBox"
${HELM} upgrade --install netbox ${NETBOX_HELM_CHART} \
  --namespace="${NAMESPACE}" \
  --create-namespace \
  --set postgresql.enabled="false" \
  --set externalDatabase.host="netbox-db.${NAMESPACE}.svc.cluster.local" \
  --set externalDatabase.existingSecretName="netbox.netbox-db.credentials.postgresql.acid.zalan.do" \
  --set externalDatabase.existingSecretKey="password" \
  --set redis.auth.password="password" \
  --set resources.requests.cpu="125m" \
  --set resources.requests.memory="128Mi" \
  --set resources.limits.cpu="1000m" \
  --set resources.limits.memory="1Gi" \
  --set redis.image.repository="bitnamilegacy/redis" \
  --set redis.master.resourcesPreset="none" \
  --set redis.master.resources.requests.cpu="50m" \
  --set redis.master.resources.requests.memory="64Mi" \
  --set redis.replica.replicaCount=0 \
  --set valkey.auth.password="password" \
  --set valkey.primary.resourcesPreset="none" \
  --set valkey.primary.resources.requests.cpu="50m" \
  --set valkey.primary.resources.requests.memory="64Mi" \
  --set valkey.replica.replicaCount=0 \
  --set global.security.allowInsecureImages=true \
  --set worker.enabled=false \
  --set dbWaitDebug=true \
  --set-json "logging=${NETBOX_LOGGING_JSON}" \
  "${PROBE_ARGS[@]}" \
    $REGISTRY_ARG

if [[ "${NETBOX_SCALED_DOWN}" == "true" ]]; then
    log "Scaling NetBox back up"
    ${KUBECTL} scale deployment netbox -n "${NAMESPACE}" --replicas=1
fi

# The helm charts for NetBox v4+ print the app version themselves, the v3.7.8 one does not
if [[ "${VERSION}" == "3.7.8" ]] ;then
    # Print the app version of the NetBox helm release
    # For the helm charts for Netbox v4+ it is printed by the helm install command
    # but not for the helm chart for v3.7.8
    log "NetBox version of the installed helm release:"
    ${HELM} list -n "${NAMESPACE}" -o json | jq -r '.[] | select(.name=="netbox") | .app_version'
fi

if [[ "$FORCE_NETBOX_NGINX_IPV4" == "true" ]]; then
  log "Creating nginx-unit ConfigMap and patching deployment"

  ${KUBECTL} apply -f "$SCRIPT_DIR/nginx-unit-config.yaml" -n "$NAMESPACE"

  # a strategic merge patch merges volumes/volumeMounts by name, so re-running does not duplicate them
  NETBOX_CONTAINER="$(${KUBECTL} get deployment netbox -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].name}')"
  ${KUBECTL} patch deployment netbox -n "$NAMESPACE" --type=strategic -p='{
    "spec": {
      "template": {
        "spec": {
          "volumes": [
            {
              "name": "unit-config",
              "configMap": {
                "name": "nginx-unit-config"
              }
            }
          ],
          "containers": [
            {
              "name": "'"${NETBOX_CONTAINER}"'",
              "volumeMounts": [
                {
                  "mountPath": "/etc/unit/nginx-unit.json",
                  "subPath": "nginx-unit.json",
                  "name": "unit-config"
                }
              ]
            }
          ]
        }
      }
    }
  }'

  # Cleanup old ReplicaSets after NetBox deployment patch to prevent volume Multi-Attach errors
  log "Cleaning up outdated NetBox ReplicaSets"
  DEPLOYMENT_NAME="netbox"
  RS_JSON=$(${KUBECTL} get rs -n $NAMESPACE -l app.kubernetes.io/component=netbox -o json | sed '/^{/,$!d')

  LATEST_RS=$(echo "$RS_JSON" | jq -r --arg DEPLOYMENT "$DEPLOYMENT_NAME" '
    .items
    | map(select(.metadata.ownerReferences[]?.kind == "Deployment" and .metadata.ownerReferences[]?.name == $DEPLOYMENT))
    | sort_by(.metadata.creationTimestamp)
    | last
    | .metadata.name
  ')

  log "Current (latest) ReplicaSet is: $LATEST_RS"

  # Delete older ones
  echo "$RS_JSON" | jq -r --arg DEPLOYMENT "$DEPLOYMENT_NAME" --arg LATEST "$LATEST_RS" '
    .items
    | map(select(
        .metadata.ownerReferences[]?.kind == "Deployment"
        and .metadata.ownerReferences[]?.name == $DEPLOYMENT
        and .metadata.name != $LATEST
      ))
    | .[].metadata.name
  ' | xargs -r -I{} ${KUBECTL} delete rs {} -n "$NAMESPACE"

  log "Forcing restart of netbox pod to reattach volume cleanly"
  ${KUBECTL} delete pods -n "$NAMESPACE" -l app.kubernetes.io/component=netbox \
      --grace-period=0 --force
fi

log "Waiting for the NetBox deployment to roll out"
${KUBECTL} rollout status --namespace="${NAMESPACE}" deployment netbox

log "Creating the ConfigMap for the local-data loader script"
TMP_CONFIGMAP_YAML="$(mktemp)"

GENERATED_FILES+=("$TMP_CONFIGMAP_YAML")
kubectl create configmap netbox-loader-script \
  --namespace="${NAMESPACE}" \
  --from-file=main.py="$SCRIPT_DIR/load-local-data-job/main.py" \
  --dry-run=client -o yaml > "$TMP_CONFIGMAP_YAML"

${KUBECTL} apply -f "$TMP_CONFIGMAP_YAML" --namespace="${NAMESPACE}"

log "Preparing the local-data load job manifest"
JOB_YAML="$SCRIPT_DIR/load-local-data-job/netbox-load-local-data-job.yaml"
TMP_JOB_YAML="$(mktemp)"
GENERATED_FILES+=("$TMP_JOB_YAML")
cp "$JOB_YAML" "$TMP_JOB_YAML"

# Internal NetBox service endpoint (used in Kind)
NETBOX_API_URL="http://netbox.${NAMESPACE}.svc.cluster.local"

PATCHED_TMP_JOB_YAML="$(mktemp)"
GENERATED_FILES+=("$PATCHED_TMP_JOB_YAML")

yq eval -o=json "$TMP_JOB_YAML" | jq \
  --arg netboxApi "$NETBOX_API_URL" \
  --arg pypiUrl "${PYPI_REPOSITORY_URL:-}" \
  --arg artifactoryHost "${ARTIFACTORY_TRUSTED_HOST:-}" \
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
' | yq eval -P - > "$PATCHED_TMP_JOB_YAML"

mv "$PATCHED_TMP_JOB_YAML" "$TMP_JOB_YAML"

# A completed Job cannot be updated in place, so replace it on every run
${KUBECTL} delete job netbox-load-local-data --namespace="${NAMESPACE}" --ignore-not-found

log "Loading local data into NetBox"
${KUBECTL} apply -n "${NAMESPACE}" -f "$TMP_JOB_YAML"
${KUBECTL} wait --namespace="${NAMESPACE}" --timeout=600s --for=condition=complete job/netbox-load-local-data

log "Done"

