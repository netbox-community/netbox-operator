#!/bin/bash

#!/bin/bash

# Define the patch payload
patch_payload='
[
    {
        "op": "add",
        "path": "/rules/-",
        "value": {
            "apiGroups": [""],
            "resources": ["secrets"],
            "verbs": ["get", "list", "watch"]
        }
    }
]
'

# Apply the patch using kubectl
kubectl --context kind-kind patch clusterrole netbox-operator-manager-role --type json -p "${patch_payload}"
