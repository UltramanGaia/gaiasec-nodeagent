# Integration Testing Guide for Container Information Collection

## Overview

This document provides comprehensive instructions for testing the container information collection feature in gaiasec-nodeagent. Tests require a containerized environment (Docker, containerd, or CRI-O).

## Prerequisites

### Required Software

- **Go**: 1.24.3 or higher
- **Docker**: Latest stable version
- **Containerd** (optional): For CRI testing
- **CRI-O** (optional): For CRI testing
- **Make**: For running test scripts

### System Permissions

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Enable containerd socket access
sudo chmod 660 /run/containerd/containerd.sock
sudo chgrp $USER /run/containerd/containerd.sock

# Enable CRI-O socket access
sudo chmod 660 /run/crio/crio.sock
sudo chgrp $USER /run/crio/crio.sock
```

## Test Environment Setup

### Option 1: Docker Environment (Recommended)

```bash
# 1. Start test containers
docker run -d --name test-nginx nginx:alpine
docker run -d --name test-redis redis:alpine
docker run -d --name test-busybox busybox sleep 3600

# 2. Verify containers are running
docker ps

# Expected output:
# CONTAINER ID   IMAGE          COMMAND                  STATUS
# abc123         nginx:alpine   "/docker-entrypoint.…"  Up 2 minutes
# def456         redis:alpine   "docker-entrypoint.…"  Up 2 minutes
# ghi789         busybox        "sleep 3600"             Up 2 minutes
```

### Option 2: Kubernetes Environment (Advanced)

```bash
# 1. Create test namespace
kubectl create namespace gaiasec-test

# 2. Deploy test pods
kubectl run test-nginx -n gaiasec-test --image=nginx:alpine
kubectl run test-redis -n gaiasec-test --image=redis:alpine
kubectl run test-busybox -n gaiasec-test --image=busybox -- sleep 3600

# 3. Verify pods are running
kubectl get pods -n gaiasec-test
```

## Running Unit Tests

### Prerequisites

```bash
# Install test dependencies
cd gaiasec-nodeagent
go mod tidy
```

### Running All Unit Tests

```bash
# Run all unit tests
go test -v ./pkg/container/... -tags unit

# Expected output:
# === RUN   TestToProtobufContainer
# --- PASS: TestToProtobufContainer (0.00s)
# === RUN   TestDockerClient_convertNetworks
# --- PASS: TestDockerClient_convertNetworks (0.00s)
# === RUN   TestDockerClient_convertPorts
# --- PASS: TestDockerClient_convertPorts (0.00s)
# === RUN   TestDockerClient_convertMounts
# --- PASS: TestDockerClient_convertMounts (0.00s)
# PASS
# ok      gaiasec-nodeagent/pkg/container          0.123s
```

### Running Specific Test Suites

```bash
# Test container type conversions
go test -v ./pkg/container -run TestToProtobufContainer

# Test Docker client conversions
go test -v ./pkg/container/runtime -run TestDockerClient_

# Test CRI client conversions
go test -v ./pkg/container/runtime -run TestCRIClient_
```

### Running Tests with Coverage

```bash
# Generate coverage report
go test -v ./pkg/container/... -coverprofile=coverage.out -tags unit

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open coverage report in browser
firefox coverage.html
```

### Expected Test Coverage

Target coverage: **80%+** for production code

| Package | Target Coverage | Current Coverage |
|---------|----------------|-------------------|
| pkg/container/types.go | 85% | - |
| pkg/container/container.go | 80% | - |
| pkg/container/runtime/docker.go | 75% | - |
| pkg/container/runtime/cri.go | 75% | - |
| pkg/container/runtime/namespace.go | 70% | - |

## Running Integration Tests

### Step 1: Build NodeAgent

```bash
cd gaiasec-nodeagent

# Clean build artifacts
go clean -cache -modcache -testcache

# Build nodeagent
go build -v -o nodeagent ./cmd/nodeagent

# Verify build
./nodeagent -version
```

### Step 2: Start Mock WebSocket Server

```bash
# Create simple WebSocket server for testing
cat > mock-server.py << 'EOF'
import asyncio
import websockets
import json

async def handler(websocket):
    print("Client connected")
    try:
        async for message in websocket:
            print(f"Received message: {message[:100]}...")

            # Parse incoming message
            data = json.loads(message)

            # Send CONTAINER_REQUEST
            if data.get("type") == "register":
                request = {
                    "type": "CONTAINER_REQUEST",
                    "destination": "node-1",
                    "source": "server",
                    "session": "test-session"
                }
                await websocket.send(json.dumps(request))
                print("Sent CONTAINER_REQUEST")

    except websockets.exceptions.ConnectionClosed:
        print("Client disconnected")

async def main():
    async with websockets.serve(handler, "localhost", 9000):
        print("WebSocket server running on ws://localhost:9000")
        await asyncio.Future()  # Run forever

asyncio.run(main())
EOF

# Start mock server in background
python3 mock-server.py &
WS_PID=$!

# Wait for server to start
sleep 2
```

### Step 3: Run NodeAgent

```bash
# Start nodeagent
./nodeagent \
  -project 1 \
  -server ws://localhost:9000 \
  -node-id test-node-1

# Expected log output:
# INFO[0000] GaiaSec Node Agent v1.0.0
# INFO[0000] ProjectID: 1
# INFO[0000] NodeID: test-node-1
# INFO[0000] Connected to WebSocket server
# INFO[0000] handleContainerRequest: received container collection request
# INFO[0000] Found 1 docker clients
# INFO[0000] Connected to Docker Engine API
# INFO[0000] Found 3 containers in docker
# INFO[0000] Collected 3 containers from Docker
# INFO[0000] handleContainerRequest: sending container info, total 3 containers
```

### Step 4: Verify Container Information

The WebSocket server should receive a `CONTAINER_RESPONSE` message with the following structure:

```json
{
  "type": "CONTAINER_RESPONSE",
  "destination": "server",
  "source": "test-node-1",
  "session": "test-session",
  "data": {
    "containers": [
      {
        "id": "abc123...",
        "name": "test-nginx",
        "status": "running",
        "image": "nginx:alpine",
        "imageId": "sha256:...",
        "runtime": "docker",
        "runtimePath": "/var/run/docker.sock",
        "networks": [
          {
            "name": "bridge",
            "ipAddress": "172.17.0.2",
            "macAddress": "02:42:ac:11:00:02"
          }
        ],
        "ports": [],
        "mounts": [],
        "labels": {},
        "annotations": {}
      },
      {
        "id": "def456...",
        "name": "test-redis",
        "status": "running",
        "image": "redis:alpine",
        "runtime": "docker",
        "networks": [...]
      },
      {
        "id": "ghi789...",
        "name": "test-busybox",
        "status": "running",
        "image": "busybox",
        "runtime": "docker",
        "networks": [...]
      }
    ]
  }
}
```

### Step 5: Cleanup

```bash
# Stop nodeagent
pkill -f nodeagent

# Stop mock WebSocket server
kill $WS_PID

# Remove test containers
docker rm -f test-nginx test-redis test-busybox
```

## Automated Testing with Make

### Create Makefile

```makefile
.PHONY: test-unit test-integration test-all clean test-coverage

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v ./pkg/container/... -tags unit -coverprofile=coverage.out
	go tool cover -func=coverage.out

# Run integration tests (requires containers)
test-integration:
	@echo "Setting up test environment..."
	docker run -d --name test-nginx nginx:alpine
	docker run -d --name test-redis redis:alpine
	@echo "Running integration tests..."
	go test -v ./pkg/container/... -tags integration
	@echo "Cleaning up test environment..."
	docker rm -f test-nginx test-redis

# Run all tests
test-all: test-unit test-integration

# Generate coverage report
test-coverage:
	@echo "Generating coverage report..."
	go test -v ./pkg/container/... -tags unit -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	go clean -cache -modcache -testcache
	rm -f coverage.out coverage.html
```

### Run Automated Tests

```bash
# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run all tests
make test-all

# Generate coverage report
make test-coverage

# Clean up
make clean
```

## Performance Testing

### Test with Large Number of Containers

```bash
# Create 100 test containers
for i in {1..100}; do
  docker run -d --name "test-$i" nginx:alpine
done

# Measure collection time
time go test -v ./pkg/container -run TestGetContainerListPerformance

# Expected: < 2 seconds for 100 containers

# Clean up
for i in {1..100}; do
  docker rm -f "test-$i"
done
```

### Memory Usage Testing

```bash
# Run nodeagent with memory profiling
go build -o nodeagent ./cmd/nodeagent
./nodeagent -project 1 -server ws://localhost:9000 &
NODEAGENT_PID=$!

# Monitor memory usage
while true; do
  ps -p $NODEAGENT_PID -o rss,vsz,pmem,comm
  sleep 1
done

# Expected memory usage: < 100MB for typical workloads

# Cleanup
kill $NODEAGENT_PID
```

## Security Testing

### Test with Unprivileged User

```bash
# Create unprivileged user
sudo useradd -m -s /bin/bash testuser
sudo usermod -aG docker testuser

# Switch to test user
su - testuser

# Test container collection
cd /pathway/to/gaiasec-nodeagent
./nodeagent -project 1 -server ws://localhost:9000

# Verify no errors in logs
```

### Test with Restricted Socket Access

```bash
# Make Docker socket read-only (should fail)
sudo chmod 444 /var/run/docker.sock

# Run nodeagent (should log error)
./nodeagent -project 1 -server ws://localhost:9000

# Expected error: "permission denied" or "connection refused"

# Restore permissions
sudo chmod 666 /var/run/docker.sock
```

## Troubleshooting

### Issue: "permission denied" connecting to Docker

**Solution**:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Issue: "no such host" or "connection refused"

**Solution**: Verify WebSocket server is running and accessible
```bash
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
  http://localhost:9000/ws/agent
```

### Issue: No containers found

**Solution**: Verify containers are running
```bash
docker ps
docker info
```

### Issue: Build fails with missing dependencies

**Solution**:
```bash
cd gaiasec-nodeagent
go mod tidy
go mod download
```

### Issue: Tests fail with "skip" messages

**Reason**: Tests are skipped when container runtime is not available

**Solution**: Run tests in a containerized environment or run full integration tests

## Continuous Integration

### GitHub Actions Example

```yaml
name: Container Collection Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      docker:
        image: docker:dind
        options: >-
          --privileged
          -e DOCKER_TLS_CERTDIR=/certs
          --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock

    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.24.3

      - name: Start test containers
        run: |
          docker run -d --name test-nginx nginx:alpine
          docker run -d --name test-redis redis:alpine

      - name: Run unit tests
        run: |
          cd gaiasec-nodeagent
          go mod tidy
          go test -v ./pkg/container/... -tags unit -coverprofile=coverage.out

      - name: Run integration tests
        run: |
          cd gaiasec-nodeagent
          go test -v ./pkg/container/... -tags integration

      - name: Upload coverage
        uses: codecov/codecov-action@v2
        with:
          file: ./gaiasec-nodeagent/coverage.out

      - name: Cleanup
        run: |
          docker rm -f test-nginx test-redis
```

## Test Checklist

### Unit Tests ✅

- [ ] TestToProtobufContainer - basic container conversion
- [ ] TestDockerClient_convertNetworks - network conversion
- [ ] TestDockerClient_convertPorts - port mapping conversion
- [ ] TestDockerClient_convertMounts - mount point conversion
- [ ] TestCRIClient_convertNetworks - CRI network conversion
- [ ] TestCRIClient_convertAnnotations - annotation handling

### Integration Tests ✅

- [ ] Docker client connects successfully
- [ ] Collects all running containers
- [ ] Returns correct container IDs and names
- [ ] Includes network configuration
- [ ] Includes port mappings
- [ ] Includes mount points
- [ ] Includes labels and annotations
- [ ] WebSocket handler responds to CONTAINER_REQUEST
- [ ] CONTAINER_RESPONSE message is valid protobuf

### Performance Tests ✅

- [ ] Collection time < 500ms for 100 containers
- [ ] Memory usage < 100MB for typical workloads
- [ ] No memory leaks after repeated collections

### Security Tests ✅

- [ ] Works with unprivileged user (docker group)
- [ ] Handles permission errors gracefully
- [ ] No sensitive data logged

---

**Last Updated**: 2026-02-26
**Test Coverage Target**: 80%+
**Reference**: Elkeid Container Collection Tests
