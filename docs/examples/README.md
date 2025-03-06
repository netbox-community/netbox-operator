# NetBox Operator Examples

This folder shows some examples how the NetBox Operator can be used. The demo environment can be prepared with the 'docs/examples/set-up/prepare-demo-env.sh' script, which creates two kind clusters with NetBox Operator and [kro] installed. One one of the clusters a NetBox instance is installed which is available to both NetBox Operator deployments.

[kro]: https://github.com/kro-run/kro/

Prerequisites:
- go version v1.24.0+
- docker image netbox-operatore:build-local
- kustomize version v5.5.0+
- kubectl version v1.32.2+
- kind v0.27.0
- docker cli

The following chapters show some examples which depend on each other.

# 0. Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Point your kubeconfig to the kind cluster 'kind-prod' `kubectl config use-context kind-prod`
2. Port-forward NetBox: `kubectl port-forward deploy/netbox 8080:8080 -n default`
3. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
4. Create a new prefix '3.0.4.8/29' with custom fields `environment: "Production", poolName: "Pool 1"`,tenant `MY_TENANT` and site `DM-Akron`

# 1. Claim a Prefix

1. Point your kubeconfig to the kind cluster 'kind-dev' `kubectl config use-context kind-dev`
2. Apply  the manifest defining the prefix claim `kubectl apply -f docs/examples/example1/netbox_v1_prefixclaim.yaml`
3. Check that the prefix claim CR got a prefix addigned `kubectl get pxc`

# 2. Claim a Prefix for a Podinfo Deployment and Create a MetalLB IPAddressPool

This example uses [kro] to map the claimed prefix to a MetalLB IPAddressPool. The required resource graph definitions and kro were installed with the set-up script.

1. create the namespace where podinfo should be deployed `kubectl create ns test`
2. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply -k docs/examples/example2 -n test`
3. check if the frontend service got an external ip address assigned `kubectl get svc podinfo -n test`

# 3. Dynamically Claim a Prefix with a Parent Prefix Selector for a Podinfo Deployment and Create a MetalLB IPAddressPool

1. create the namespace where podinfo should be deployed `kubectl create ns int`
2. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply -k docs/examples/example3 -n int`
3. check if the frontend service got an external ip address assigned `kubectl get svc podinfo -n int`

# 4. Mutli cluster set up with source of truth in netbox

1. Point your kubeconfig to the kind cluster 'kind-dev' `kubectl config use-context kind-prod`
2. create the namespace where podinfo should be deployed `kubectl create ns instance1`
3. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply -k docs/examples/example4/instance1 -n instance1`
4. check if the frontend service got an external ip address assigned `kubectl get svc podinfo -n instance1`
5. create the namespace where podinfo should be deployed `kubectl create ns instance2`
6. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply -k docs/examples/example4/instance2 -n instance2`
7. check if the frontend service got an external ip address assigned `kubectl get svc podinfo -n instance2`