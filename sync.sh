#!/bin/bash
POD_ID=$(kubectl -n sothoth get pods |grep api-gateway|awk '{print $1}' | head -n 1)
kubectl -n sothoth cp ./agent/java.zip ${POD_ID}:/usr/share/nginx/html/plugins/microinsight/
