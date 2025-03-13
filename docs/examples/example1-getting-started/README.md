# Example 1: Getting Started

# 0. Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox: `kubectl port-forward --context kind-london deploy/netbox 8080:8080 -n default`
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom fields 'environment: prod'

# 1.1 Claim a Prefix

1. Apply  the manifest defining the prefix claim `kubectl apply --context kind-zurich -f docs/examples/example1-getting-started/simple_prefixclaim.yaml`
2. Check that the prefix claim CR got a prefix addigned `kubectl get --context kind-zurich pxc,px -w`

![Example 1.1](simple_prefixclaim.drawio.svg)

# 1.2 Dynamically Claim a Prefix with a Parent Prefix Selector

1. create the namespace where podinfo should be deployed `kubectl create  --context kind-zurich ns int`
2. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply --context kind-zurich  -f docs/examples/example1-getting-started/simple_prefixclaim.yaml`
3. check if the frontend service got an external ip address assigned `kubectl get --context pxc,px -w`

![Example 1.2](dynamic-prefixclaim.drawio.svg)

# 1.3 Claim a Prefix for a Podinfo Deployment and Create a MetalLB IPAddressPool

This example uses [kro] to map the claimed prefix to a MetalLB IPAddressPool. The required resource graph definitions and kro were installed with the set-up script.

1. create the namespace where podinfo should be deployed `kubectl create  --context kind-zurich ns test`
2. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply --context kind-zurich -k docs/examples/example1-getting-started -n test`
3. check if the frontend service got an external ip address assigned `kubectl get --context kind-zurich svc podinfo -n test`


![Example 1.3](metallb-ipaddresspool-netbox.drawio.svg)
