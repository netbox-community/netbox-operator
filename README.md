# NetBox Operator

**Disclaimer:** This project is currently under development and may change rapidly, including breaking changes. Use with caution in production environments.

NetBox Operator extends the Kubernetes API by allowing users to manage NetBox resources – such as IP addresses and prefixes – directly through Kubernetes. This integration brings Kubernetes-native features like reconciliation, ensuring that network configurations are maintained automatically, thereby improving both efficiency and reliability.

![Figure 1: NetBox Operator High-Level Architecture](docs/netbox-operator-high-level-architecture.drawio.svg)

# Getting Started

## Prerequisites

- go version v1.23.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- kind
- docker cli

# How to use NetBox Operator

## Running both NetBox Operator and NetBox on a local kind cluster

Note: This requires Docker BuildKit.

- Create kind cluster with a NetBox deployment: `make create-kind`
- Deploy the NetBox Operator on the local kind cluster: `make deploy-kind` (In case you're using podman use `CONTAINER_TOOL=podman make deploy-kind`)

To optionally access the NetBox UI:

- Port-forward NetBox: `kubectl port-forward deploy/netbox 8080:8080 -n default`
- Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`, you will be able to access the local NetBox instance running in the kind cluster.

## Testing NetBox Operator using samples

In the folder `config/samples/` you can find example manifests to create IpAddress, IpAddressClaim, Prefix, and PrefixClaim resources. Apply them to the cluster with `kubectl apply -f <file-name>` and use your favorite Kubernetes tools to display.

Example of assigning a Prefix using PrefixClaim:

![Figure 2: PrefixClaim example with a NetBox and NetBox Operator instance deployed on the same cluster](docs/prefixclaim-sample-with-netbox-running-in-cluster.drawio.svg)

1. Apply a PrefixClaim: `kubectl apply -f config/samples/netbox_v1_prefixclaim.yaml`
2. Wait for ready condition: `kubectl wait prefix prefixclaim-sample --for=condition=Ready`
3. List PrefixClaim and Prefix resources: `kubectl get pxc,px`
4. In the prefix status fields you’ll be able to see the netbox URL of the resource. Login with the default `admin`/`admin` credentials to access the NetBox resources.

Key information can be found in the yaml formatted output of these resources, as well as in the events and Operator logs.

# Restoration from NetBox

In the case that the cluster containing the NetBox Custom Resources managed by this NetBox Operator is not backed up (e.g. using Velero), we need to be able to restore some information from NetBox. This includes two mechanisms implemented in this NetBox Operator:

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

We cordially invite collaboration from the community to enhance the quality and functionality of this project. Whether you are addressing bugs, introducing new features, refining documentation, or assisting with items on our to-do list, your contributions are highly valued and greatly appreciated. Please take a look at [Contribution guide] for more details.

[Contribution guide]: ./CONTRIBUTING.md

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
