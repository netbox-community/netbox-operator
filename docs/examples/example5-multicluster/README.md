# Example 5: Advanced Feature Multi Cluster Support

## Introduction

NetBox Operator uses NetBox to avoid IP overlaps. This means that we can use NetBox Operator on multiple clusters. You can try this out using the example in this directory.

This example shows how to claim multiple prefixes from different clusters and make them available as metalLB ip address pools.

### 0.1 Create a local cluster with nebox-installed

1. set up your local environment to run the following examples with the set up script 'docs/examples/example5-multicluster/prepare-demo-env.sh'

### 0.2 Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox:
```bash
kubectl port-forward deploy/netbox 8080:8080
```
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom field 'environment: prod'

### 0.3 Navigate to the example folder

Navigate to 'docs/examples/example5-multicluster/' to run the examples below

## Example Steps

1. Create ip address pools on the london cluster
```bash
kubectl apply --context kind-london -f docs/examples/example5-multicluster/london-pools.yaml
```
2. Create ip address pool on the zurich cluster
```bash
kubectl create --context kind-zurich -f docs/examples/example5-multicluster/zurich-pools.yaml
```
3. Look up the created prefix claims
```bash
kubectl get --context kind-london pxc -A
```
and
```bash
kubectl get --context kind-zurich pxc -A
```

![Example 2](multicluster.drawio.svg)