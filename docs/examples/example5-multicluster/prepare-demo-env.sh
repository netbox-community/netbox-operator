#!/bin/bash
set -e
#create the kind clusters zurich and london
./docs/examples/example5-multicluster/create-kind-clusters.sh zurich london

# install netbox in the london cluster and load demo data
kubectl config use-context kind-london
./kind/deploy-netbox.sh london "4.1.10" default

# install NetBox Operator
kubectl config use-context kind-london
kind load docker-image netbox-operator:build-local --name london
kind load docker-image netbox-operator:build-local --name london  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
kustomize build docs/examples/example5-multicluster/ | kubectl apply -f -

kubectl config use-context kind-zurich
kind load docker-image netbox-operator:build-local --name zurich
kind load docker-image netbox-operator:build-local --name zurich  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
kustomize build docs/examples/example5-multicluster/ | kubectl apply -f -
kind load docker-image curlimages/curl --name zurich
kind load docker-image curlimages/curl --name zurich
kubectl run curl --image curlimages/curl --image-pull-policy=Never -- sleep infinity

# expose netbox service
kubectl config use-context kind-london
kubectl apply -f docs/examples/example5-multicluster/netbox-svc.yaml
kubectl apply -f docs/examples/example5-multicluster/netbox-l2advertisement.yaml
