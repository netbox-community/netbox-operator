# Example 2: Multi Cluster

This example shows how to claim multiple prefixes from different clusters and make them available as metalLB ip address pools.

1. Create ip address pools on the london cluster
```bash
kubectl apply --context kind-london -f docs/examples/example2-multicluster/london-pools.yaml
```
2. Create ip address pool on the zurich cluster
```bash
kubectl create --context kind-zurich -f docs/examples/example2-multicluster/zurich-pools.yaml
```
3. Look up the created prefix claims and metalLB ipaddresspools
```bash
watch kubectl get --context kind-london pxc,ipaddresspools -A
```
and
```bash
watch kubectl get --context kind-zurich pxc,ipaddresspools -A
```

![Example 2](multicluster.drawio.svg)
