#!/bin/bash
set -e

NAMESPACE="${NAMESPACE:-default}"

echo "Checking for gaiasec-api-gateway pod in namespace '${NAMESPACE}'..."

if ! command -v kubectl &> /dev/null; then
    echo "kubectl not found. Skipping sync."
    exit 0
fi

POD_ID=$(kubectl -n "${NAMESPACE}" get pods 2>/dev/null | grep -E "gaiasec-api-gateway.*Running" | awk '{print $1}' | head -n 1)

if [ -z "$POD_ID" ]; then
    echo "No running gaiasec-api-gateway pod found in namespace '${NAMESPACE}'. Skipping sync."
    echo "To sync to a pod, ensure the gateway is running and set NAMESPACE if needed."
    exit 0
fi

echo "Found pod: ${POD_ID}"

echo "Syncing linux/amd64 binary..."
kubectl -n "${NAMESPACE}" cp ./agent/nodeagent-linux-amd64 "${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/"

echo "Syncing linux/arm64 binary..."
kubectl -n "${NAMESPACE}" cp ./agent/nodeagent-linux-arm64 "${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/"

echo "Verifying sync..."
kubectl -n "${NAMESPACE}" exec -it "${POD_ID}" -- ls -la /usr/share/nginx/html/plugins/nodeagent/

echo "Sync completed successfully."
