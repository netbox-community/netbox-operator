# Example 2: Glue NetBox CRs to MetalLB CRs

## Introduction

So we have Prefixes represented as Kubernetes Resources. Now what can we do with this?

We use kro.io to glue this to MetalLB IPAddressPools

1. Apply the kro resource graph definition, defining the mapping from the prefix claim to the metalLB ip address pool
```bash
kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-netbox.yaml
```
2. Apply the manifests to create a deployment with a service and a metallb-ip-address-pool-netbox to create a metalLB IPAddressPool from the prefix claimed from NetBox
```bash
kubectl apply --context kind-zurich -f docs/examples/example1-getting-started/ip-address-pool.yaml
```
3. Apply the manifests to createa deployment with a service that gets a ip assigned from the metalLB pool created in the prevoius step
```bash
kubectl apply --context kind-zurich -k docs/examples/example1-getting-started/sample-deployment.yaml
```
4. check if the prefixclaim and ipaddresspool got created
```bash
watch kubectl get --context kind-zurich pxc,ipaddresspools my-nginx -A
```
5. check if the service got an external ip address assigned
```bash
watch kubectl get --context kind-zurich svc my-nginx -n nginx
```


![Example 1.3](metallb-ipaddresspool-netbox.drawio.svg)
