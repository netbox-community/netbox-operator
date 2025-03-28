#!/bin/bash
set -e -u -o pipefail

NAMESPACE=""
VERSION="4.1.8" # default value
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

./kind/deploy-netbox.sh kind $VERSION $NAMESPACE
