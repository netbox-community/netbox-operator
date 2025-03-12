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

## Installation methods

There are several ways to install NetBox Operator on your Cluster:

- NetBox Operator Helm Chart. More information: <https://github.com/netbox-community/netbox-chart/>
- Manifests in <config/default> of this repo. Make sure to first set the correct image using `kustomize edit set image controller=<correctimage>`. Images can be found on the Releases page on GitHub
- The Makefile targets in this repository.
- For debugging and developing, please read further below.

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

# Mixed usage of Prefixes

Note that NetBox does handle the Address management of Prefixes separately from IP Ranges and IP Addresses. This is important to know when you plan to use the same NetBox Prefix as a parentPrefix for your IpAddressClaims, IpRangeClaims and PrefixClaims.

Example:

- Assume you use an existing empty NetBox Prefix "192.168.0.0/24" as the `.spec.parentPrefix` for your PrefixClaims, IpAddressClaims and IpRangeClaims.
- You create a PrefixClaim with `.spec.prefixLength` of `/25`, NetBox Operator will assign "192.168.0.0/25"
- You create a IpAddressClaim, NetBox Operator will assign "192.168.0.1/32". Important: NetBox ignores the Prefix in that case and will not return "192.168.0.129" as the first available IP!
- You create a IpAddressClaim with `.spec.size` of `2`, NetBox Operator will assign "192.168.0.2/32" to "192.168.0.3/32". Important: NetBox ignores the Prefix in that case and will not return "192.168.0.129" as the first available IP!


This means that you need to plan your automation carefully. As a rule of thumb:
- If you plan mixed use of the same parentPrefix for both single IPs (/32s) as well as Prefixes (non /32s), use PrefixClaims for everything and avoid using IpRangeClaims and IpAddressClaims.
- If you are in full control of a Prefix and you know it will only be used for assigning IP Addresses and IP Ranges, you can use IpAddressClaims and IpRangeClaims.
- If you don't know what the parentPrefix is used for, avoid using IpAddressClaims and IpRangeClaims.

The same applies if you use parentPrefixSelector with PrefixClaims. The above example is IPv4 based but will be the same with IPv6 equivalents.

# Restoration from NetBox

In the case that the cluster containing the NetBox Custom Resources managed by this NetBox Operator is not backed up (e.g. using Velero), we need to be able to restore some information from NetBox. This includes two mechanisms implemented in this NetBox Operator:

- `IpAddressClaim` and `PrefixClaim` have the flag `preserveInNetbox` in their spec. If set to true, the NetBox Operator will not delete the assigned IP Address/Prefix in NetBox when the Kubernetes Custom Resource is deleted
- In NetBox, a custom field (by default `netboxOperatorRestorationHash`) is used to identify an IP Address/Prefix based on data from the IpAddressClaim/PrefixClaim resource

Use Cases for this Restoration:

- Disaster Recovery: In case the cluster is lost, IP Addresses can be restored with the IPAddressClaim only
- Sticky IPs: Some services do not handle changes to IPs well. This ensures the IP/Prefix assigned to a Custom Resource is always the same.

# `ParentPrefixSelector` in `PrefixClaim`

Please read [ParentPrefixSelector guide] for more information!

[ParentPrefixSelector guide]: ./ParentPrefixSelectorGuide.md

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

# Monitoring

When the operator is deployed with the default kustomization (located at config/default/) the metrics endpoint is already exposed and provides the [default kubebuilder metrics].

[default kubebuilder metrics]: https://book.kubebuilder.io/reference/metrics-reference.

For the monitoring of the state of the CRs reconciled by the operator [kube state metrics] can be used, check the kube-state-metrics documentation for instructions on configuring it to collect metrics from custom resources.

[kube state metrics]: https://github.com/kubernetes/kube-state-metrics

# FAQ

## What is the difference between `.spec.customFields` and `.spec.parentPrefixSelector`?

`.spec.customFields` are the NetBox Custom Fields assigned to the resource in NetBox.
`.spec.parentPrefixSelector` is used by a Claim Controller (e.g. the controller of PrefixClaim) to find a suitable Prefix to get e.g. a Prefix from.

## What is the difference between `.spec.tenant` and `.spec.parentPrefixSelector.tenant`?

`.spec.tenant` is the tenant that is assigned to the resource in NetBox.
`.spec.parentPrefixSelector.tenant` is used by a Claim Controller (e.g. the controller of PrefixClaim) to find a suitable Prefix to get e.g. a Prefix from.

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
