#!/bin/bash
set -e

# install netbox in the london cluster and load demo data
make deploy-kind

# install curl pod to demo access to created service
kubectl run curl --image curlimages/curl -- sleep infinity

DEPLOYMENT_NAME=netbox-operator-controller-manager
NAMESPACE=netbox-operator-system
CONTEXT=kind-kind

# install MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.8/config/manifests/metallb-native.yaml

# install kro
helm install kro oci://ghcr.io/kro-run/kro/kro \
  --namespace kro \
  --create-namespace \
  --version=0.2.1

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
kubectl apply --context $CONTEXT -f docs/examples/2-load-balancer-ip/load-balancer-ip-pool-netbox.yaml
