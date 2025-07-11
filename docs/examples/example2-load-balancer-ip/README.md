# Example 2: Glue NetBox CRs to MetalLB CRs

## Introduction

So we have Prefixes represented as Kubernetes Resources. Now what can we do with this?

We use kro.run to glue this to MetalLB IPAddressPools

### 0.1 Create a local cluster with nebox-installed

1. use the 'create-kind' and 'deploy-kind' targets from the Makefile to create a kind cluster and deploy NetBox and NetBox Operator on it
```bash
make create-kind
make deploy-kind
```

### 0.2 Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox:
```bash
kubectl port-forward deploy/netbox 8080:8080
```
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom field 'environment: prod'

### 0.3 Navigate to the example folder

Navigate to 'docs/examples/example2-load-balancer-ip/' to run the examples below

## Example Steps

0. Install kro and metallb with the installation script `docs/examples/example2-load-balancer-ip/prepare-demo-env.sh`
Then navigate to 'docs/examples/example2-load-balancer-ip' to follow the steps below.

1. Inspect the spec of the sample prefix claim CR
```bash
cat zurich-pool.yaml
```
2. Apply the manifests to create a deployment with a service and a metallb-ip-address-pool-netbox to create a metalLB IPAddressPool from the prefix claimed from NetBox
```bash
kubectl apply -f zurich-pool.yaml
```
3. Check if the prefixclaim CR and the metalLB ipaddresspool CR got created
```bash
kubectl get pxc,ipaddresspool -A
```
4. Inspect the spec of the sample prefix claim CR
```bash
cat sample-deployment.yaml
```
5. Apply the manifests to create deployment with a service that gets a ip assigned from the metalLB pool created in the previous step
```bash
kubectl apply -f sample-deployment.yaml
```
6. check if the service got an external ip address assigned and that the nginx deployment is ready
```bash
kubectl get deploy,svc -n nginx
```
7. try to connect to your service with the external ip
```bash
k exec curl -it -- sh
curl <external-ip>
```


![Example 2](metallb-ipaddresspool-netbox.drawio.svg)
