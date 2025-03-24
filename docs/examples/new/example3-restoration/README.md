# Example 3: restoration Feature Restoration

## Introduction

NetBox Operator offers a few restoration features. In this example we showcase how NetBox Operator can restore prefixes. This is especially useful when e.g. you need sticky IPs or Prefixes when redeploying an entire cluster.

## Instructions

First, let's create some resources we want to restore later.

```bash
kubectl create ns restoration
kubectl -n restoration apply -f kro-rdg-poolfromnetbox.yaml
kubectl -n restoration wait --for=condition=Ready prefixclaims --all
kubectl -n restoration get prefixclaims
```

![Figure 4: Restoration](restore.drawio.svg)

Since we set `.spec.preserveInNetbox` to `true`, we can delete and restore the resources. To delete all reasources, delete the entire namespace:

```bash
kubectl delete ns restoration
```

Make sure the resources are gone in Kubernetes:

```bash
kubectl -n restoration get prefixclaims
```

Verify in the NetBox UI that the Prefixes still exist.

Now apply the manifests again and verify they become ready.

```bash
kubectl create ns restoration
kubectl -n restoration apply -f kro-rdg-poolfromnetbox.yaml
kubectl -n restoration wait --for=condition=Ready prefixclaims --all
kubectl -n restoration get prefixclaims
```

Note that the assigned Prefixes are the same as before. You can also play around with this by just restoring single prefixes. If you're curious about how this is done, make sure to read [the "Restoration from NetBox" section in the main README.md](https://github.com/netbox-community/netbox-operator/tree/main?tab=readme-ov-file#restoration-from-netbox) and to check out the code. Also have a look at the "Netbox Restoration Hash" custom field in NetBox.
