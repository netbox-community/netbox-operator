#!/bin/bash

# This script creates the specified number of kind clusters with MetalLB
set -e
# Define colors
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if cluster names are provided
if [ $# -eq 0 ]; then
  echo -e "${RED}Error: No cluster names provided${NC}"
  echo "Usage: $0 <clustername1> <clustername2> ... <clusternameN>"
  exit 1
fi

number_of_clusters=$1
clustername=${2:-dns}  # Set default cluster name to 'dns' if not provided
mkdir -p tmp
i=0

# Loop to create the specified number of clusters
for clustername in "$@"; do
  config_file="docs/examples/set-up/cluster-cfg.yaml"
  temp_config="tmp/cluster-$clustername-cfg.yaml"
  i=$((i + 1))

  if [ -f "$config_file" ]; then
    # Make a temporary copy of the configuration file
    cp "$config_file" "$temp_config"

    # Modify apiServerPort in the copied config file
    sed -i'' -e "s/apiServerPort: 6443/apiServerPort: $((6443 + i))/g" "$temp_config"
    rm "$temp_config"-e

    # check if cluster exists
    if kind get clusters | grep -q "^${clustername}$"; then
      echo "Cluster ${clustername} already exists. Skipping creation."
      continue
    fi
    kind create cluster --name $clustername --config $temp_config || { echo -e "${RED}Error: Failed to create cluster ${clustername}${NC}"; rm -f "$temp_config"; exit 1; }

    # install MetalLB
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.8/config/manifests/metallb-native.yaml
  else
    echo -e "${RED}Error: Configuration file $config_file not found${NC}"
    exit 1
  fi
done
