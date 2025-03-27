# Example 4: Advanced Feature Multi Cluster Support

## Introduction

NetBox Operator uses NetBox to avoid IP overlaps. This means that we can use NetBox Operator on multiple clusters. You can try this out using the example in this directory.

This example shows how to claim multiple prefixes from different clusters and make them available as metalLB ip address pools.

Navigate to 'docs/examples/example2-krm-glue' to run the following commands.

0. Install kro and metallb with the installation script `docs/examples/example5-multicluster/prepare-demo-env.sh`
Then navigate to 'docs/examples/edocs/examples/example5-multicluster' to follow the steps below.

1. Create ip address pools on the london cluster
```bash
kubectl apply --context kind-london -f docs/examples/example2-multicluster/london-pools.yaml
```
2. Create ip address pool on the zurich cluster
```bash
kubectl create --context kind-zurich -f docs/examples/example2-multicluster/zurich-pools.yaml
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