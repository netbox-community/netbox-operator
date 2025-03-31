# Example 3: restoration Feature Restoration
```bash
kubectl create ns restoration

kubectl -n restoration apply -f prefixclaim-restore1.yaml
kubectl -n restoration apply -f prefixclaim-restore2.yaml
kubectl -n restoration apply -f prefixclaim-restore3.yaml
kubectl -n restoration wait --for=condition=Ready prefixclaims --all
kubectl -n restoration get prefixclaims
```

```bash
kubectl delete ns restoration
```

```bash
kubectl create ns restoration

kubectl -n restoration apply -f prefixclaim-restore3.yaml
kubectl -n restoration wait --for=condition=Ready prefixclaims --all
kubectl -n restoration get prefixclaims

kubectl -n restoration apply -f prefixclaim-restore2.yaml
sleep 1
kubectl -n restoration apply -f prefixclaim-restore1.yaml
kubectl -n restoration wait --for=condition=Ready prefixclaims --all
kubectl -n restoration get prefixclaims
```

Delete Leases to speed up:

```bash
kubectl -n netbox-operator-system get lease -oname | grep -v netbox | xargs -n1 kubectl -n netbox-operator-system delete
```
