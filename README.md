# NetBox Operator

**Disclaimer:** This project is currently under development and may change rapidly, including breaking changes. Use with caution in production environments.

NetBox Operator extends the Kubernetes API by allowing users to manage NetBox resources – such as IP addresses and prefixes – directly through Kubernetes. This integration brings Kubernetes-native features like reconciliation, ensuring that network configurations are maintained automatically, thereby improving both efficiency and reliability.

![Figure 1: NetBox Operator High-Level Architecture](docs/netbox-operator-high-level-architecture.drawio.svg)

# Getting Started

## Prerequisites

- go version v1.22.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- kind
- docker cli

## Running and developing locally

### Running the NetBox Operator and NetBox on a local kind cluster

Note: This requires Docker BuildKit.

- Create kind cluster with a NetBox deployment: `make create-kind`
- Port-forward NetBox: `kubectl port-forward deploy/netbox 8080:8080 -n default`
- Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`, you will be able to access the local NetBox instance running in the kind cluster.
- Deploy the NetBox Operator on the local kind cluster: `make deploy-kind` (In case you're using podman use `CONTAINER_TOOL=podman make deploy-kind`)

### Running the NetBox Operator on your machine and NetBox on a local kind cluster

- Create kind cluster with a NetBox deployment: `make create-kind`
- Port-forward NetBox: `kubectl port-forward deploy/netbox 8080:8080 -n default`
- Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`, you will be able to access the local NetBox instance running in the kind cluster.
- Open a new terminal window and export the following environment variables:

    ```bash
    export NETBOX_HOST="localhost:8080"
    export AUTH_TOKEN="0123456789abcdef0123456789abcdef01234567"
    export POD_NAMESPACE="default"
    export HTTPS_ENABLE="false"
    export NETBOX_RESTORATION_HASH_FIELD_NAME="netboxOperatorRestorationHash"
    ```

- Run the NetBox Operator locally `make install && make run`

### Running the NetBox Operator on your machine using an existing NetBox and Kubernetes cluster

Note: This requires a running NetBox instance that you can use (e.g. <https://demo.netbox.dev>) as well as a kubernetes cluster (can be as simple as `kind create cluster`)

- Prepare NetBox (based on the demo NetBox instance):
  - Open <https://demo.netbox.dev/plugins/demo/login/> and create any user
  - Open <https://demo.netbox.dev/user/api-tokens/> and create a token "0123456789abcdef0123456789abcdef01234567" with default settings
  - Open <https://demo.netbox.dev/extras/custom-fields/add/> and create a custom field called "netboxOperatorRestorationHash" for Object types "IPAM > IP Address" and "IPAM > Prefix"
- Open a new terminal window and export the following environment variables:

    ```bash
    export NETBOX_HOST="demo.netbox.dev"
    export AUTH_TOKEN="0123456789abcdef0123456789abcdef01234567"
    export POD_NAMESPACE="default"
    export HTTPS_ENABLE="true"
    export NETBOX_RESTORATION_HASH_FIELD_NAME="netboxOperatorRestorationHash"
    ```

- Run the NetBox Operator locally `make install && make run`

## Testing NetBox Operator using samples

In the folder `config/samples/` you can find example manifests to create IpAddress, IpAddressClaim, Prefix and PrefixClaim resources. Apply them to the cluster with `kubectl apply -f <file-name>` and use your favorite Kubernetes tools to displa.

Example of assigning a Prefix using PrefixClaim:

![Figure 2: PrefixClaim example with a NetBox and NetBox Operator instance deployed on the same cluster](docs/prefixclaim-sample-with-netbox-running-in-cluster.drawio.svg)

1. Apply a PrefixClaim: `kubectl apply -f config/samples/netbox_v1_prefixclaim.yaml`
2. Wait for ready condition: `kubectl wait prefix prefixclaim-sample --for=condition=Ready`
3. List PrefixClaim and Prefix resources: `kubectl get pxc,px`
4. For the prefix you’ll be able to see the URL, you can open it (login is admin/admin) to browse through the NetBox resources.

Make sure to also discover the yaml output of these resources, check the events and Operator logs.

## To Deploy on the cluster

### Build and push your image to the location specified by `IMG`

```sh
make docker-build docker-push IMG=<some-registry>/netbox-operator:tag
```

> **NOTE**: This image ought to be published in the personal registry you specified, and it is required to have access to pull the image from the working environment. Make sure you have the proper permission to the registry if the above commands don’t work.

### Install the CRDs into the cluster

```sh
make install
```

### Deploy the Manager to the cluster with the image specified by `IMG`

```sh
make deploy IMG=<some-registry>/netbox-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

### Create instances of your solution

You can apply the samples (examples) from the config/sample directory:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples have default values to test it out.

## To Uninstall

### Delete the instances (CRs) from the cluster

```sh
kubectl delete -k config/samples/
```

### Delete the APIs(CRDs) from the cluster

```sh
make uninstall
```

### UnDeploy the controller from the cluster

```sh
make undeploy
```

## Restoration from NetBox

In case the cluster that contains the NetBox Custom Resources managed by this NetBox Operator is not backed up (e.g. using Velero), we need to be able to restore some information from NetBox. This includes two mechanisms implemented in this NetBox Operator:

- `IpAddressClaim` and `PrefixClaim` have the flag `preserveInNetbox` in their spec. If set to true, the NetBox Operator will not delete the assigned IP Address/Prefix in NetBox when the Kubernetes Custom Resource is deleted
- In NetBox, a custom field (by default `netboxOperatorRestorationHash`) is used to identify an IP Address/Prefix based on data from the IpAddressClaim/PrefixClaim resource

Use Cases for this Restoration:

- Disaster Recovery: In case the cluster is lost, IP Addresses can be restored with the IPAddressClaim only
- Sticky IPs: Some services do not handle changes to IPs well. This ensures the IP/Prefix assigned to a Custom Resource is always the same.

# Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/netbox-operator:tag
```

> **NOTE**: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run `kubectl apply -f <URL for YAML BUNDLE>` to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/netbox-operator/<tag or branch>/dist/install.yaml
```

# Contributing

We cordially invite collaboration from the community to enhance the quality and functionality of this project. Whether you are addressing bugs, introducing new features, refining documentation, or assisting with items on our to-do list, your contributions are highly valued and greatly appreciated.

> **NOTE**: Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

# License

Copyright 2024 Swisscom (Schweiz) AG.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
