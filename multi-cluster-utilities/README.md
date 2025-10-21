# Multicluster Configuration
The create-kubeconfig-secret.sh is copied from [multicluster-runtime project](https://github.com/kubernetes-sigs/multicluster-runtime/tree/main), especially, from the `Kubeconfig Provider Example`.

This Readme cover only what is relevant for setting up a `Management cluster` which runs the controller, and `Resource Clusters` which host the Netbox Operator Resources. For more information over the scripts, and how a multicluster setup could be setup with Kubeconfig provider please read [this](https://github.com/kubernetes-sigs/multicluster-runtime/blob/main/examples/kubeconfig/README.md)

## 1. Create Management Cluster

Follow the guide in project's root folder.
If all the prerequisites are in place, then `make create-kind` creates the cluster which is going to be used for the all the controller's depedencies (netbox backend, databases etc).
This cluster also contains RBAC for handling the netbox relates CRs, but those RBACs should not be necessary for our example.

## 2. Create Resource Clusters

Create your Resouce Clusters & Provision Netbox-Operator Custom Resource Definitions. For each resource cluster you want to create, execute:

- `kind create cluster --name <res-cluster-name>`
- `make install`

Sidenote: kind create command changes the default context in which the kubectl commands point to. When you are done from this step, the kubectl context will be set at the last cluster you created.

## 3. Establish cross-cluster access

Set kubectl context back to `management cluster`
`kubectl config use-context kind-kind`

Execute RBAC scripts for kubeconfig provider, towards each cluster you created in the previous step.
- `./create-kubeconfig-secret.sh -c <res-cluster-name>`

For each cluster that gets configures as a 'Resource' cluster, a secret is populated in the 'Management' cluster.
Make sure that the appropriate secrets are populated in the kind-kind cluster, with names `kind-<resource cluster name>`.

## Limitations
Currently the controller could be executed locally, not from the managment cluster.
In order to make it executable from the management cluster:
- ClusterRole `manager-role` needs to allow reading, listing and watching secrets.
- The secrets generated from the `create-kubeconfig-secret` needs to point to the correct ip. currently it's pointing a localhost ip.
