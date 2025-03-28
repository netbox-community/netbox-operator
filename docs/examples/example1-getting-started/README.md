# Example 1: Getting Started

# 0.1 Create a local cluster with nebox-installed

1. use the 'create-kind' and 'deploy-kind' targets from the Makefile to create a kind cluster and deploy NetBox and NetBox Operator on it
```bash
make create-kind
make deploy-kind
```

# 0.2 Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox:
```bash
kubectl port-forward deploy/netbox 8080:8080
```
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom field 'environment: prod'

# 0.3 Navigate to the example folder

Navigate to 'docs/examples/example1-getting-started' to run the examples below

# 1.1 Claim a Prefix

In this example, we use a `.spec.parentPrefix` that we know in advance. This is useful if you already know exactly from which prefix you want to claim from.

1. Inspect the spec of the sample prefix claim CR
```bash
cat prefixclaim-simple.yaml
```
2. Apply  the manifest defining the prefix claim
```bash
kubectl apply  -f prefixclaim-simple.yaml
```
3. Check that the prefix claim CR got a prefix assigned
```bash
kubectl get  pxc,px
```

![Example 1.1](prefixclaim-simple.drawio.svg)

# 1.2 Dynamically Claim a Prefix with a Parent Prefix Selector

In this example, we use a `.spec.parentPrefixSelector`, which is a list of selectors that tell NetBox Operator from which parent prefixes to claim our Prefix from.

Navigate to 'docs/examples/example1-getting-started' to run the following commands.

1. Inspect the spec of the sample prefix claim CR
```bash
cat prefixclaim-dynamic.yaml
```
2. Apply  the manifest defining the prefix claim
```bash
kubectl apply   -f prefixclaim-dynamic.yaml
```
3. Check that the prefix claim CR got a prefix addigned
```bash
kubectl get  pxc,px
```

![Example 1.2](prefixclaim-dynamic.drawio.svg)
