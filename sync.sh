#!/bin/bash
POD_ID=$(kubectl -n sothoth get pods |grep api-gateway|awk '{print $1}' | head -n 1)
kubectl -n sothoth cp ./agent/nodeagent-linux-amd64 ${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/
kubectl -n sothoth cp ./agent/nodeagent-linux-arm64 ${POD_ID}:/usr/share/nginx/html/plugins/nodeagent/
kubectl -n sothoth exec -it ${POD_ID} -- ls -la /usr/share/nginx/html/plugins/nodeagent/
