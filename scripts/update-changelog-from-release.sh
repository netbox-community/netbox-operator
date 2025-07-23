#!/bin/bash
set -e

# Script to update CHANGELOG.md from GitHub release content
# Usage: ./update-changelog-from-release.sh <tag_name>

TAG_NAME=$1

if [ -z "$TAG_NAME" ]; then
    echo "Usage: $0 <tag_name>"
    exit 1
fi

echo "Updating CHANGELOG.md for release $TAG_NAME..."

# Get release data from GitHub
RELEASE_DATA=$(gh release view "$TAG_NAME" --json tagName,publishedAt,body,url)

# Extract fields
TAG=$(echo "$RELEASE_DATA" | jq -r '.tagName')
DATE=$(echo "$RELEASE_DATA" | jq -r '.publishedAt' | cut -d'T' -f1)
BODY=$(echo "$RELEASE_DATA" | jq -r '.body')
URL=$(echo "$RELEASE_DATA" | jq -r '.url')

# Create the changelog entry
ENTRY="## [$TAG] - $DATE

$BODY

[Full Release]($URL)

---
"

# Create a temporary file with the new entry
TEMP_FILE=$(mktemp)

# Check if CHANGELOG.md exists
if [ ! -f "CHANGELOG.md" ]; then
    echo "Creating CHANGELOG.md..."
    cat > CHANGELOG.md << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF
fi

# Read the header (everything before the first ## entry)
awk '/^## \[/ {exit} {print}' CHANGELOG.md > "$TEMP_FILE"

# Add the new entry
echo "$ENTRY" >> "$TEMP_FILE"

# Add the rest of the changelog
awk '/^## \[/ {found=1} found {print}' CHANGELOG.md >> "$TEMP_FILE"

# Replace the original file
mv "$TEMP_FILE" CHANGELOG.md

echo "CHANGELOG.md updated successfully!"