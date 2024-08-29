# netbox-operator

# Getting Started

## Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

## Running and developing locally with the Go debugger

### Using a kind cluster

Follow the instructions to bring up a locally running kind cluster, with NetBox and a NetBox operator running within it:
- execute `make create-kind`
- set the following environment varialbes:
    ```
    NETBOX_HOST="localhost:8080"
    AUTH_TOKEN="0123456789abcdef0123456789abcdef01234567"
    POD_NAMESPACE="default"
    HTTPS_ENABLE="false"
    NETBOX_RESTORATION_HASH_FIELD_NAME="netboxOperatorRestorationHash"
    ```
- execute `make deploy-kind`
- in a separate terminal, run `kubectl port-forward deploy/netbox 8080:8080 -n default`
- go to your favorite browser and type in `localhost:8080`, with the username `admin` and password `admin`, you will be able to access the local NetBox instance running in the kind cluster

### Using an existing NetBox instance and Kubernetes cluster

Prerequisites:
- a NetBox instance to test against
- a Kubernetes cluster with the netbox-operator CRDs installed (point the kubeconfig to the cluster and run `make install`)

There are some mandatory environment variables to set to run the netbox-operator locally: `NETBOX_HOST`, `AUTH_TOKEN` and `POD_NAMESPACE`.

If you want to run the operator against a host with HTTPS protocol you need to set the `HTTPS_ENABLE=true`.

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

In case the cluster that contains the NetBox Custom Resources managed by this Operator is not backed up (e.g. using Velero), we need to be able to restore some information from NetBox. This includes two mechanisms implemented in this Operator:
- `IpAddressClaim` and `PrefixClaim` have the flag `preserveInNetbox` in their spec. If set to true, the Operator will not delete the assigned IP Address/Prefix in NetBox when the Kubernetes Custom Resource is deleted
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
// TODO(user): Add detailed information on how you would like others to contribute to this project

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
