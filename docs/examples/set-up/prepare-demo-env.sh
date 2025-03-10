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
# install resource graph defintions
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox-parent-prefix.yaml
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox.yaml


kubectl config use-context kind-zurich
kind load docker-image netbox-operator:build-local --name zurich
kind load docker-image netbox-operator:build-local --name zurich  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
kustomize build docs/examples/set-up/ | kubectl apply -f -
# install resource graph defintions
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox-parent-prefix.yaml
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox.yaml
