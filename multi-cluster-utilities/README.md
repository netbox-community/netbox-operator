# Multicluster Configuration
The create-kubeconfig-secret.sh is copied from [multicluster-runtime project](https://github.com/kubernetes-sigs/multicluster-runtime/tree/main), especially, from the `Kubeconfig Provider Example`.

This Readme cover only what is relevant for setting up a `Management cluster` which hosts the kubernetes operator, and `Resource Clusters` which host the Netbox Operator Resources. For more information over the scripts, and how a multicluster setup could be setup with Kubeconfig provider please read [this](https://github.com/kubernetes-sigs/multicluster-runtime/blob/main/examples/kubeconfig/README.md)

## 1. Create Management Cluster

Follow the guide in project's root folder.
If all the prerequisites are in place, then `make create-kind` creates the cluster which is going to be used for the all the controller's depedencies (netbox backend, databases etc).
This cluster is also configured with the netbox-operator CRDs but the CRs are hosted and reconciled only in Resource Clusters.

## 2. Create Resource Clusters

Create your Resouce Clusters & Provision Netbox-Operator Custom Resource Definitions. For each resource cluster you want to create, execute:

- `kind create cluster --name <res-cluster-name>`
- `make install`

Sidenote: kind create command changes the default context in which the kubectl commands point to. When you are done from this step, the kubectl context will be set at the last cluster you created.

## 3. Establish cross-cluster access

Set kubectl context back to `management cluster`
`kubectl config use-context kind-kind`

Execute RBAC scripts for kubeconfig provider, towards each cluster you created in the previous step.
- `./create-kubeconfig-secret.sh -c kind-<res-cluster-name>`

For each cluster that gets configured as a 'Resource' cluster, a secret is populated in the 'Management' cluster.
Make sure that the appropriate secrets are populated in the kind-kind cluster, with names `kind-<res-cluster-name>`.

## 4-1. Execute manager process locally
At this point, you should be able to sucesfully start netbox operator process locally after:
- Establishing a port forward from Management cluster to your host for the Netbox service: `kubectl port-forward deploy/netbox 8080:8080 -n default`
- Setting environment variable `export NETBOX_HOST=localhost:8080`

## 4-2. Execute manager process on managment cluster
Deploying the manager in the management cluster involves additional manual steps.

From the project's parent directory, execute `make deploy-kind`

### Patch cluster role of netbox operator manager
ClusterRole `manager-role` needs to allow reading, listing and watching secrets.
    - Execute `patch-netbox-clusterrole.sh`

### Update secret in controller cluster
The secrets generated from the `create-kubeconfig-secret` needs to reffer to the correct ip:port for each resource cluster. Currently it's pointing a localhost ip, which is only reachable from the host machine.
- Execute script `create-kubeconfig-secret-cluster.sh -c kind-<res-cluster-name> --skip-create-rbac`
    - This script updates the secret on management cluster, to use the IP of control-plane node of the resource cluster, retrieved from ``docker inspect <resouce-cluster>-control-plane | jq '.[0].NetworkSettings.Networks.kind.IPAddress'``
    - The port of the K8s API server is assumed to be `6443`. You can check it with `docker inspect <resource-cluster>-control-plane | jq '.[0].NetworkSettings.Ports'`

## 5. Test Reconciliation
Apply an example CR in resource cluster and check if it's getting reconciled.
`kubectl --context <resource-cluster> apply -f config/samples/netbox_v1_ipaddress.yaml`