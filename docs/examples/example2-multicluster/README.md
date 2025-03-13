# Example 3: Multi Cluster

1. create the namespace where podinfo should be deployed `kubectl create --context kind-london ns instance1`
2. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply --context kind-london -k docs/examples/example4/instance1 -n instance1`
3. check if the frontend service got an external ip address assigned `kubectl get --context kind-london svc podinfo -n instance1`
4. create the namespace where podinfo should be deployed `kubectl create --context kind-london ns instance2`
5. Install podinfo with with the kustomization and apply the instance of the resource graph definition to claim a prefix and create the MetalLB IPAddressPool `kubectl apply --context kind-london -k docs/examples/example4/instance2 -n instance2`
6. check if the frontend service got an external ip address assigned `kubectl get --context kind-london svc podinfo -n instance2`
