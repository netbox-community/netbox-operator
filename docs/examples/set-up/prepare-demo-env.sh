#!/bin/bash
set -e
# create the kind clusters netbox, prod and dev
./docs/examples/set-up/create-kind-clusters.sh prod dev

# install netbox in the netbox cluster and load demo data
kubectl config use-context kind-prod
./kind/deploy-netbox.sh prod "4.1.8" default
kubectl apply -f docs/examples/set-up/netbox-svc.yaml
kubectl apply -f docs/examples/set-up/netbox-l2advertisement.yaml


# install NetBox Operator
kubectl config use-context kind-prod
kind load docker-image netbox-operator:build-local --name prod
kustomize build docs/examples/set-up/ | kubectl apply -f -
# install resource graph defintions
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox-parent-prefix.yaml
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox.yaml


kubectl config use-context kind-dev
kind load docker-image netbox-operator:build-local --name dev
kustomize build docs/examples/set-up/ | kubectl apply -f -
# install resource graph defintions
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox-parent-prefix.yaml
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-from-netbox.yaml
