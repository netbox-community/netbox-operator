# Example 1: Getting Started

# 0. Ma![img.png](img.png)nually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox:
```bash
kubectl port-forward --context kind-london deploy/netbox 8080:8080
```
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom field 'environment: prod'

# 1.1 Claim a Prefix

In this example, we use a `.spec.parentPrefix` that we know in advance. This is useful if you already know exactly from which prefix you want to claim from.

1. Apply  the manifest defining the prefix claim
```bash
kubectl apply --context kind-zurich -f docs/examples/example1-getting-started/prefixclaim-simple.yaml
```
2. Check that the prefix claim CR got a prefix addigned
```bash
watch kubectl get --context kind-zurich pxc,px
```

![Example 1.1](prefixclaim-simple.drawio.svg)

# 1.2 Dynamically Claim a Prefix with a Parent Prefix Selector

In this example, we use a `.spec.parentPrefixSelector`, which is a list of selectors that tell NetBox Operator from which parent prefixes to claim our Prefix from.

1. Apply  the manifest defining the prefix claim
```bash
kubectl apply --context kind-zurich  -f docs/examples/example1-getting-started/prefixclaim-dynamic.yaml
```
2. Check that the prefix claim CR got a prefix addigned
```bash
watch kubectl get --context kind-zurich pxc,px
```

![Example 1.2](prefixclaim-dynamic.drawio.svg)
