# Mini Kubernetes Clone

A simplified Kubernetes implementation in Go for learning purposes. This project implements core Kubernetes components including API Server, Scheduler, Kubelet, and a CLI tool.

## Features

- ✅ **API Server** (Fiber-based) - Central management point with REST API
- ✅ **Scheduler** - Assigns pods to nodes using round-robin algorithm
- ✅ **Kubelet** - Manages containers on nodes via Docker
- ✅ **Controller Manager** - Monitors cluster health
- ✅ **CLI Tool (minictl)** - kubectl-like interface
- ✅ **In-memory Storage** - Thread-safe pod and node storage
- ✅ **Docker Integration** - Real container management

## Architecture

```
┌──────────┐
│ minictl  │
└────┬─────┘
     │
     ▼
┌────────────────┐
│  API Server    │ (Port 8080)
│  (Fiber)       │
└───┬────────┬───┘
    │        │
    ▼        ▼
┌──────┐  ┌────────────┐
│Sched-│  │Controller  │
│uler  │  │Manager     │
└──┬───┘  └────────────┘
   │
   ▼
┌──────────┐
│ Kubelet  │ → Docker
└──────────┘
```

## Prerequisites

- Go 1.21 or higher
- Docker installed and running
- Linux, macOS, or Windows with WSL2

## Installation

### 1. Clone and Setup

```bash
git clone https://github.com/sohel902833/minikube-clone.git
cd minikube-clone
go mod download
```

### 2. Build All Components

```bash
# Using Makefile (recommended)
make build

# Or manually
go build -o bin/apiserver cmd/apiserver/main.go
go build -o bin/scheduler cmd/scheduler/main.go
go build -o bin/kubelet cmd/kubelet/main.go
go build -o bin/minictl cmd/minictl/main.go
```

## Quick Start

### Option 1: Automated Startup (Recommended)

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Start entire cluster
./scripts/start-all.sh

# The script will:
# - Build binaries if needed
# - Start API Server on port 8080
# - Start Scheduler
# - Start Kubelet (node1)
# - Verify cluster health
```

### Option 2: Manual Startup

**Terminal 1 - API Server:**

```bash
make run-api
# or
./bin/apiserver --port 8080
```

**Terminal 2 - Scheduler:**

```bash
make run-scheduler
# or
./bin/scheduler --api-server http://localhost:8080
```

**Terminal 3 - Kubelet:**

```bash
make run-kubelet
# or
./bin/kubelet --node-name node1 --api-server http://localhost:8080
```

## Usage Examples

### 1. Check Cluster Status

```bash
# List nodes
./bin/minictl get nodes

# Output:
# NAME    STATUS    AGE
# node1   Running   2m30s
```

### 2. Create a Pod

```bash
# Apply pod from manifest
./bin/minictl apply -f manifests/example-pod.yaml

# Output:
# Pod nginx-pod created successfully
```

### 3. List Pods

```bash
./bin/minictl get pods

# Output:
# NAME         STATUS    NODE     AGE
# nginx-pod    Running   node1    1m15s
```

### 4. Get Pod Details

```bash
./bin/minictl describe pod nginx-pod

# Output:
# Name:         nginx-pod
# Namespace:    default
# Status:       Running
# Node:         node1
# Created:      2024-01-15T10:30:00Z
#
# Containers:
#   nginx:
#     Image: nginx:latest
#     Ports: 80
```

### 5. Delete a Pod

```bash
./bin/minictl delete pod nginx-pod

# Output:
# pod nginx-pod deleted successfully
```

### 6. View Logs (Basic)

```bash
./bin/minictl logs nginx-pod
```

## Testing the Cluster

### Create Multiple Pods

```bash
# Create example pods
./bin/minictl apply -f manifests/example-pod.yaml

# Verify pods are running
./bin/minictl get pods

# Check node status
./bin/minictl get nodes
```

### Using cURL to Test API

```bash
# Health check
curl http://localhost:8080/healthz

# List all pods
curl http://localhost:8080/api/v1/pods

# Get specific pod
curl http://localhost:8080/api/v1/pods/nginx-pod

# Create pod via API
curl -X POST http://localhost:8080/api/v1/pods \
```
