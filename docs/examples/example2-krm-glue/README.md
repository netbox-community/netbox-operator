# Example 2: Glue NetBox CRs to MetalLB CRs

## Introduction

So we have Prefixes represented as Kubernetes Resources. Now what can we do with this?

We use kro.run to glue this to MetalLB IPAddressPools

Navigate to 'docs/examples/example2-krm-glue' to run the following commands.

0. Install kro and metallb with the installation script `docs/examples/new/example2-krm-glue/prepare-demo-env.sh`
Then navigate to 'docs/examples/example2-krm-glue' to follow the steps below.

1. Inspect the spec of the sample prefix claim CR
```bash
cat kro-rdg-poolfromnetbox.yaml
```
2. Apply the manifests to create a deployment with a service and a metallb-ip-address-pool-netbox to create a metalLB IPAddressPool from the prefix claimed from NetBox
```bash
kubectl apply -f kro-rdg-poolfromnetbox.yaml
```
3. Check if the prefixclaim CR and the metalLB ipaddresspool CR got created
```bash
kubectl get pxc,ipaddresspool -A
```
4. Inspect the spec of the sample prefix claim CR
```bash
cat sample-deployment.yaml
```
5. Apply the manifests to createa deployment with a service that gets a ip assigned from the metalLB pool created in the prevoius step
```bash
kubectl apply -f sample-deployment.yaml
```
6. check if the service got an external ip address assigned
```bash
kubectl get svc my-nginx -n nginx
```
7. try to connect to your service with the external ip
```bash
k exec curl -it -- sh
curl <external-ip>
```


![Example 2](metallb-ipaddresspool-netbox.drawio.svg)
