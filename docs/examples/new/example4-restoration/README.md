# Example 3: Advanced Feature Restoration

## Introduction

NetBox Operator offers a few advanced features. In this example we showcase how NetBox Operator can restore prefixes. This is especially useful when e.g. you need sticky IPs or Prefixes when redeploying an entire cluster.

## Instructions

![Figure 4: Restoration](restore.drawio.svg)

Since we set `.spec.preserveInNetbox` to `true`, we can delete and restore the resources. To delete all reasources, delete the entire namespace:

```bash
kubectl --context kind-london delete ns advanced
```

Make sure the resources are gone in Kubernetes:

```bash
kubectl --context kind-london -n advanced get prefixclaims
```

Verify in the NetBox UI that the Prefixes still exist.

Now apply the manifests again and verify they become ready.

```bash
kubectl --context kind-london create ns advanced
kubectl --context kind-london apply -f kro-rdg-poolfromnetbox.yaml
kubectl --context kind-london -n advanced wait --for=condition=Ready prefixclaims --all
kubectl --context kind-london -n advanced get prefixclaims
```

Note that the assigned Prefixes are the same as before. You can also play around with this by just restoring single prefixes. If you're curious about how this is done, make sure to read [the "Restoration from NetBox" section in the main README.md](https://github.com/netbox-community/netbox-operator/tree/main?tab=readme-ov-file#restoration-from-netbox) and to check out the code. Also have a look at the "Netbox Restoration Hash" custom field in NetBox.
