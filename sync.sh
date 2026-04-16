#!/bin/bash
set -e

echo "Checking for gaiasec-api-gateway pod"

if ! command -v kubectl &> /dev/null; then
    echo "kubectl not found. Skipping sync."
    exit 0
fi

POD_ID=$(kubectl get pods 2>/dev/null | grep -E "gaiasec-api-gateway.*Running" | awk '{print $1}' | head -n 1)

if [ -z "$POD_ID" ]; then
    echo "No running gaiasec-api-gateway pod found. Skipping sync."
    exit 0
fi

echo "Found pod: ${POD_ID}"

echo "Syncing linux/amd64 binary..."
kubectl cp ./agent/nodeagent-linux-amd64 "${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/"

echo "Syncing linux/arm64 binary..."
kubectl cp ./agent/nodeagent-linux-arm64 "${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/"

echo "Verifying sync..."
kubectl exec -it "${POD_ID}" -- ls -la /usr/share/nginx/html/plugins/nodeagent/

echo "Sync completed successfully."
