#!/bin/bash
set -e
# create the kind clusters zurich and london
./docs/examples/set-up/create-kind-clusters.sh zurich london

# install netbox in the london cluster and load demo data
kubectl config use-context kind-london
./kind/deploy-netbox.sh london "4.1.8" default
kubectl apply -f docs/examples/set-up/netbox-svc.yaml
kubectl apply -f docs/examples/set-up/netbox-l2advertisement.yaml

# install NetBox Operator
kubectl config use-context kind-london
kind load docker-image netbox-operator:build-local --name london
kind load docker-image netbox-operator:build-local --name london  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
kustomize build docs/examples/set-up/ | kubectl apply -f -


kubectl config use-context kind-zurich
kind load docker-image netbox-operator:build-local --name zurich
kind load docker-image netbox-operator:build-local --name zurich  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
kustomize build docs/examples/set-up/ | kubectl apply -f -

DEPLOYMENT_NAME=netbox-operator-controller-manager
NAMESPACE=netbox-operator-system
CONTEXT=kind-london

while true; do
  # Check if the deployment is ready
  READY_REPLICAS=$(kubectl --context $CONTEXT get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
  DESIRED_REPLICAS=$(kubectl --context $CONTEXT get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.status.replicas}')

  if [[ "$READY_REPLICAS" == "$DESIRED_REPLICAS" ]] && [[ "$READY_REPLICAS" -gt 0 ]]; then
    echo "Deployment $DEPLOYMENT_NAME in cluster $CONTEXT is ready."
    break
  else
    echo "Waiting... Ready replicas in cluster $CONTEXT: $READY_REPLICAS / $DESIRED_REPLICAS"
    sleep 5
  fi
done
kubectl apply --context $CONTEXT -f docs/examples/set-up/metallb-ip-address-pool-netbox.yaml

CONTEXT=kind-zurich
while true; do
  # Check if the deployment is ready
  READY_REPLICAS=$(kubectl --context $CONTEXT get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
  DESIRED_REPLICAS=$(kubectl --context $CONTEXT get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.status.replicas}')

  if [[ "$READY_REPLICAS" == "$DESIRED_REPLICAS" ]] && [[ "$READY_REPLICAS" -gt 0 ]]; then
    echo "Deployment $DEPLOYMENT_NAME in cluster $CONTEXT is ready."
    break
  else
    echo "Waiting... Ready replicas in cluster $CONTEXT: $READY_REPLICAS / $DESIRED_REPLICAS"
    sleep 5
  fi
done
kubectl apply --context $CONTEXT -f docs/examples/set-up/metallb-ip-address-pool-netbox.yaml
