# Example 3: Advanced Feature Prefix Exhaustion

## Introduction

NetBox Operator offers a few advanced features. In this example we showcase how NetBox Operator can recover from prefix exhaustion.

When a Prefix is exhausted and this is fixed in the NetBox backend (e.g. by the Infrastructure team), NetBox Operator will automatically reconcile this.

## Instructions

![Figure 1: Starting Point](exhaustion-1-starting-point.drawio.svg)

Create a /24 Prefix (e.g. 1.122.0.0/24) with Custom Field Environment set to "prod" in NetBox UI.

Apply Resource and show PrefixClaims:

```bash
kubectl create ns advanced
kubectl apply -f prefixclaim-exhaustion.yaml
kubectl -n advanced get prefixclaims,prefixes
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
kubectl -n advanced annotate prefixclaim prefixclaim-exhaustion-sample-3 reconcile="$(date)" --overwrite
```

Confirm that the third Prefix is now also assigned:

```bash
kubectl -n advanced get prefixclaims,prefixes
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
