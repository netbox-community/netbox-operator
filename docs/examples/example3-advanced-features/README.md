# Example 3: Advanced Features

## Description

This demo showcases the two following cases:

- Example 3a: Automatic Reconcilation in case of Prefix Exhaustion. When a Prefix is exhausted and this is fixed in the NetBox backend, NetBox Operator will automatically reconcile this.
- Example 3b: Restoration of Prefixes

## Instructions

### Example 3a: Prefix Exhaustion and Reconciliation

![Figure 1: Starting Point](exhaustion-1-starting-point.drawio.svg)

Create a /24 Prefix (e.g. 1.122.0.0/24) with Custom Field Environment set to "prod" in NetBox UI.

Apply Resource and show PrefixClaims:

```bash
kubectl --context kind-london apply -f netbox_v1_prefixclaim.yaml
kubectl --context kind-london -n advanced get prefixclaims,prefixes
```

Note that only 2 out of the 3 PrefixClaims will become Ready. This is because the /24 Prefix is exhausted already after two Prefixes. This will look similar to this (note the order is non-deterministic):

```bash
NAME                                                     PREFIX           PREFIXASSIGNED   READY   AGE
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-1   1.122.0.0/25     True             True    2m2s
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-2   1.122.0.128/25   True             True    2m2s
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-3                    False                    2m2s

NAME                                                PREFIX           READY   ID    URL                                   AGE
prefix.netbox.dev/prefixclaim-exhaustion-sample-1   1.122.0.0/25     True    148   http://172.18.1.2/ipam/prefixes/148   2m2s
prefix.netbox.dev/prefixclaim-exhaustion-sample-2   1.122.0.128/25   True    149   http://172.18.1.2/ipam/prefixes/149   2m2s
```

![Figure 2: Parent Prefix Exhausted](exhaustion-2-prefix-exhausted.drawio.svg)


Create another /24 Prefix (e.g. 1.100.0.0/24) with Custom Field Environment set to "prod" in NetBox UI.

Wait for the PrefixClaim to be reconciled again or trigger reconciliation by e.g. adding an annotation:

```bash
kubectl --context kind-london -n advanced annotate prefixclaim prefixclaim-exhaustion-sample-3 reconcile="$(date)" --overwrite
```

Confirm that the third Prefix is now also assigned:

```bash
kubectl --context kind-london -n advanced get prefixclaims,prefixes
```

Which should look as follows:

```bash
NAME                                                     PREFIX           PREFIXASSIGNED   READY   AGE
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-1   1.122.0.0/25     True             True    4s
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-2   1.122.0.128/25   True             True    4s
prefixclaim.netbox.dev/prefixclaim-exhaustion-sample-3   1.100.0.0/25     True             True    4s

NAME                                                PREFIX           READY   ID    URL                                   AGE
prefix.netbox.dev/prefixclaim-exhaustion-sample-1   1.122.0.0/25     True    148   http://172.18.1.2/ipam/prefixes/148   4s
prefix.netbox.dev/prefixclaim-exhaustion-sample-2   1.122.0.128/25   True    149   http://172.18.1.2/ipam/prefixes/149   4s
prefix.netbox.dev/prefixclaim-exhaustion-sample-3   1.100.0.0/25     True    151   http://172.18.1.2/ipam/prefixes/151   3s```
```

![Figure 3: Parent Prefix Exhaustion fixed](exhaustion-3-after-fix.drawio.svg)

### Example 3b: Restoration

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
kubectl --context kind-london apply -f netbox_v1_prefixclaim.yaml
kubectl --context kind-london -n advanced wait --for=condition=Ready prefixclaims --all
kubectl --context kind-london -n advanced get prefixclaims
```

Note that the assigned Prefixes are the same as before. You can also play around with this by just restoring single prefixes. If you're curious about how this is done, make sure to read [the "Restoration from NetBox" section in the main README.md](https://github.com/netbox-community/netbox-operator/tree/main?tab=readme-ov-file#restoration-from-netbox) and to check out the code. Also have a look at the "Netbox Restoration Hash" custom field in NetBox.
