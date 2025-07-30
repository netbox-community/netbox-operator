#!/bin/bash
set -e

# Script to bump version in kustomization.yaml
# Usage: ./bump-version.sh <new_version>

NEW_VERSION=$1

if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <new_version>"
    exit 1
fi

KUSTOMIZATION_FILE="config/manager/kustomization.yaml"

if [ ! -f "$KUSTOMIZATION_FILE" ]; then
    echo "Error: $KUSTOMIZATION_FILE not found!"
    exit 1
fi

echo "Updating version to $NEW_VERSION in $KUSTOMIZATION_FILE..."

# Update the newTag
sed -i.bak "s/newTag: .*/newTag: $NEW_VERSION/" "$KUSTOMIZATION_FILE"
rm -f "$KUSTOMIZATION_FILE.bak"

echo "Version bumped to $NEW_VERSION successfully!"
