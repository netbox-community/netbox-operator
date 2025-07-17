# e2e tests

# How to run the test locally?

- [Install](https://kyverno.github.io/chainsaw/latest/quick-start/install/) `chainsaw`: `go install github.com/kyverno/chainsaw@latest`
- Execute `make test-e2e` or `chainsaw test --namespace e2e` with specific flags (see Makefile)
- Use `--test-dir` flag if you want to execute a specific test.
- Use `--skip-delete` flag in case you want to skip the deletion of the Kubernetes resources (e.g. for debugging). Note that this results in leftovers and you might need to clean up things.

# How to add a new test locally?

- Create a clean cluster `kind delete cluster --name kind && make create-kind deploy-kind && kubectl port-forward deploy/netbox 8080:8080 -n default`
- During development, we would need to have a clean NetBox instance between runs, which we can do with the following commands as soon as we create a new cluster
    - Backup: `kubectl exec pod/netbox-db-0 -- bash -c "pg_dump --clean -U postgres netbox" > database.sql`
- The simplest test case is `tests/e2e/prefix/ipv4/prefixclaim-ipv4-apply-update`
    - We always need a `chainsaw-test.yaml`
- Perform a clean run by resetting the database first, then execute the test   
    - Reset database `cat database.sql | kubectl exec -i pod/netbox-db-0 -- psql -U postgres -d netbox`
        - Make sure that in the `e2e` namespace, no leftover CRs are there
    - Execute the entire e2e test `make test-e2e`
        - Or just perform a specific run, e.g. `chainsaw test --test-dir tests/e2e/prefix/ipv4/prefixclaim-ipv4-apply-update`

# Some debugging tips

- `kubectl get prefixclaim,prefix,ipaddressclaim,ipaddress,iprange,iprangeclaim -A`
- For monitoring failures, we can use event (which is a native k8s object)
   - e.g. `kubectl events --for prefixclaim.netbox.dev/prefixclaim-apply-prefixexhausted-3 -o yaml`
- I am not entirely sure why is resetting the database not clean enough, maybe the redis is doing some caching that I am not aware of
