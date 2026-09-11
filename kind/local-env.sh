#!/bin/bash
set -e -u -o pipefail

NAMESPACE=""
VERSION="4.4.9" # default value
CLUSTER="kind"  # name of the kind cluster
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
if kind get clusters 2> /dev/null | grep -qx "${CLUSTER}"; then
    echo "kind cluster '${CLUSTER}' already exists, reusing it"
else
    kind create cluster --name "${CLUSTER}"
fi

# everything below targets the current context, so make sure it points at the kind cluster
KUBE_CONTEXT="kind-${CLUSTER}"
if ! kubectl config use-context "${KUBE_CONTEXT}" > /dev/null; then
    echo "ERROR: no kubectl context '${KUBE_CONTEXT}', refusing to deploy into '$(kubectl config current-context)'" >&2
    exit 1
fi

kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
kubectl wait --for=jsonpath='{.status.phase}'=Active --timeout=30s namespace/${NAMESPACE}

./kind/deploy-netbox.sh "${CLUSTER}" $VERSION $NAMESPACE
