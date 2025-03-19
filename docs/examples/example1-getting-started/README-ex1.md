# Example 1: Getting Started

# 0. Manually Create a Prefix in NetBox

Before prefixes and ip addresses can be claimed with the NetBox operator, a prefix has to be created in NetBox.

1. Port-forward NetBox: `kubectl port-forward --context kind-london deploy/netbox 8080:8080`
2. Open <http://localhost:8080> in your favorite browser and log in with the username `admin` and password `admin`
3. Create a new prefix '3.0.0.64/26' with custom field 'environment: prod'

# 1.1 Claim a Prefix

1. Apply  the manifest defining the prefix claim `kubectl apply --context kind-zurich -f docs/examples/example1-getting-started/simple_prefixclaim.yaml`
2. Check that the prefix claim CR got a prefix addigned `watch -n 1 kubectl get --context kind-zurich pxc,px`

![Example 1.1](simple_prefixclaim.drawio.svg)

# 1.2 Dynamically Claim a Prefix with a Parent Prefix Selector

1. Apply  the manifest defining the prefix claim `kubectl apply --context kind-zurich  -f docs/examples/example1-getting-started/dynamic-prefix-claim.yaml`
2. Check that the prefix claim CR got a prefix addigned `watch -n 1 kubectl get --context kind-zurich pxc,px`

![Example 1.2](dynamic-prefixclaim.drawio.svg)

# 1.3 Claim a Prefix and Create a MetalLB IPAddressPool, create a depoyment which is exposed with a service using an ip from the claimed prefix

This example uses [kro] to map the claimed prefix to a MetalLB IPAddressPool. The required resource graph definitions and kro were installed with the set-up script.

1. Apply the kro resource graph definition, defining the mapping from the prefix claim to the metalLB ip address pool `kubectl apply -f docs/examples/set-up/metallb-ip-address-pool-netbox.yaml`
2. Apply the manifests to create a deployment with a service and a metallb-ip-address-pool-netbox to create a metalLB IPAddressPool from the prefix claimed from NetBox `kubectl apply --context kind-zurich -f docs/examples/example1-getting-started/ip-address-pool.yaml`
3. Apply the manifests to createa deployment with a service that gets a ip assigned from the metalLB pool created in the prevoius step. `kubectl apply --context kind-zurich -k docs/examples/example1-getting-started/sample-deployment.yaml`
4. check if the prefixclaim and ipaddresspool got created `watch -n 1 kubectl get --context kind-zurich pxc,ipaddresspools my-nginx -A`
5. check if the service got an external ip address assigned `watch -n 1 kubectl get --context kind-zurich svc my-nginx -n nginx`


![Example 1.3](metallb-ipaddresspool-netbox.drawio.svg)
