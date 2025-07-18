# How to contribute

NetBox Operator is Apache 2.0 licensed and accepts contributions via GitHub pull requests.

This document outlines the basics of contributing to NetBox Operator.

## Running and developing NetBox-Operator locally

There are several ways of deploying the NetBox operator for development.

### Running both NetBox Operator and NetBox on a local kind cluster

Note: This requires Docker BuildKit.

- Create kind cluster with a NetBox deployment: `make create-kind`
- Deploy the NetBox Operator on the local kind cluster: `make deploy-kind` (In case you're using podman use `CONTAINER_TOOL=podman make deploy-kind`)

To optionally access the NetBox UI:

- Port-forward NetBox: `kubectl port-forward deploy/netbox 8080:8080 -n default`
- Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`, you will be able to access the local NetBox instance running in the kind cluster.

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

### Running the NetBox Operator on your machine and use an existing NetBox and Kubernetes cluster

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

In the folder `config/samples/` you can find example manifests to create IpAddress, IpAddressClaim, Prefix and PrefixClaim resources. Apply them to the cluster with `kubectl apply -f <file-name>` and use your favorite Kubernetes tools to display.

Example of assigning a Prefix using PrefixClaim:

![Figure 2: PrefixClaim example with a NetBox and NetBox Operator instance deployed on the same cluster](docs/prefixclaim-sample-with-netbox-running-in-cluster.drawio.svg)

1. Apply a PrefixClaim: `kubectl apply -f config/samples/netbox_v1_prefixclaim.yaml`
2. Wait for ready condition: `kubectl wait prefix prefixclaim-sample --for=condition=Ready`
3. List PrefixClaim and Prefix resources: `kubectl get pxc,px`
4. In the prefix status fields you’ll be able to see the netbox URL of the resource. Login with the default `admin`/`admin` credentials to access the NetBox resources.

Key information can be found in the yaml formatted output of these resources, as well as in the events and Operator logs.

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

## `make` targets

Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## Running e2e tests locally

Please read the [README in the e2e test directory] for more information!

[README in the e2e test directory]: ./tests/e2e/README.md
