# Operational Manual

This document describes how to troubleshoot errors in the NetBox Operator.
Known issues are collected on the [GitHub issues](https://github.com/netbox-community/netbox-operator/issues) page.

## Troubleshooting

### Check Logs
Use the following command to get logs from the operator:

```bash
kubectl logs -n <namespace> deployment/netbox-operator-controller-manager
```

### Check CR Status
Inspect the CRs status:

```bash 
kubectl describe <netbox-crd> <netbox-cr> -n <namespace>
```

E.g.:
```bash
kubectl describe prefixclaim prefixclaim-sample -n <namespace>
kubectl describe ipaddressclaim ipaddressclaim-sample -n <namespace>
kubectl describe prefix prefix-sample -n <namespace>
kubectl describe ipaddress ipaddress-sample -n <namespace> 
```
This will show you the status of the operator and any errors it may have encountered.

### Verify Operator Version

```bash
kubectl get deployment netbox-operator-controller-manager -n <namespace> -o=jsonpath="{.spec.template.spec.containers[*].image}"
```

### Look at Related Pods
If netbox-oeprator is not running correctly, inspect the related pods:

```bash
kubectl logs deployment/netbox-operator-controller-manager -n <namespace> -c manager
kubectl describe pod <pod-name> -n <namespace>
```

### Check Events
Events might give hints about what’s going wrong:

```bash
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
```