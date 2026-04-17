#!/bin/bash
set -e

docker build -t ultramangaia/gaiasec-env:nodeagent .
docker push ultramangaia/gaiasec-env:nodeagent
