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
