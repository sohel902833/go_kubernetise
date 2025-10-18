# Mini Kubernetes Clone - Complete Folder Structure

```
minikube-clone/
│
├── cmd/                                    # Main applications
│   ├── apiserver/
│   │   └── main.go                        # API Server entry point
│   ├── scheduler/
│   │   └── main.go                        # Scheduler entry point
│   ├── kubelet/
│   │   └── main.go                        # Kubelet entry point
│   └── minictl/
│       └── main.go                        # CLI tool entry point
│
├── pkg/                                    # Shared libraries
│   ├── api/
│   │   └── handlers.go                    # API handlers with Fiber
│   ├── storage/
│   │   └── store.go                       # In-memory storage layer
│   ├── scheduler/
│   │   └── scheduler.go                   # Scheduling logic
│   ├── kubelet/
│   │   └── kubelet.go                     # Kubelet implementation
│   └── controller/
│       └── controller.go                  # Controller manager
│
├── types/
│   └── types.go                           # Core data structures
│
├── manifests/                              # Example YAML files
│   ├── example-pod.yaml                   # Sample pod definition
│   ├── nginx-pod.yaml                     # Nginx example
│   └── multi-container-pod.yaml           # Multi-container example
│
├── bin/                                    # Compiled binaries (generated)
│   ├── apiserver
│   ├── scheduler
│   ├── kubelet
│   └── minictl
│
├── scripts/                                # Utility scripts
│   ├── start-all.sh                       # Start all components
│   ├── stop-all.sh                        # Stop all components
│   └── test-cluster.sh                    # Test cluster functionality
│
├── docs/                                   # Documentation
│   ├── architecture.md                    # Architecture overview
│   ├── api-reference.md                   # API documentation
│   └── development.md                     # Development guide
│
├── configs/                                # Configuration files
│   ├── apiserver.yaml                     # API server config
│   ├── scheduler.yaml                     # Scheduler config
│   └── kubelet.yaml                       # Kubelet config
│
├── test/                                   # Test files
│   ├── integration/
│   │   ├── api_test.go
│   │   └── scheduler_test.go
│   └── unit/
│       ├── storage_test.go
│       └── types_test.go
│
├── go.mod                                  # Go module file
├── go.sum                                  # Go dependencies checksum
├── README.md                               # Project documentation
├── LICENSE                                 # License file
├── Makefile                                # Build automation
└── .gitignore                              # Git ignore rules
```

## Detailed File Descriptions

### Core Application Files (`cmd/`)

| File                    | Purpose                                                      | Lines |
| ----------------------- | ------------------------------------------------------------ | ----- |
| `cmd/apiserver/main.go` | Starts Fiber web server, initializes storage, sets up routes | ~90   |
| `cmd/scheduler/main.go` | Main loop for pod scheduling, connects to API server         | ~40   |
| `cmd/kubelet/main.go`   | Node agent initialization, Docker client setup               | ~50   |
| `cmd/minictl/main.go`   | CLI commands using Cobra framework                           | ~400  |

### Package Files (`pkg/`)

| File                           | Purpose                                       | Lines |
| ------------------------------ | --------------------------------------------- | ----- |
| `pkg/api/handlers.go`          | Fiber HTTP handlers for all REST endpoints    | ~250  |
| `pkg/storage/store.go`         | Thread-safe in-memory storage with mutex      | ~180  |
| `pkg/scheduler/scheduler.go`   | Scheduling algorithm and pod placement logic  | ~200  |
| `pkg/kubelet/kubelet.go`       | Container lifecycle management via Docker SDK | ~250  |
| `pkg/controller/controller.go` | Reconciliation loops and health checks        | ~150  |

### Type Definitions (`types/`)

| File             | Purpose                              | Lines |
| ---------------- | ------------------------------------ | ----- |
| `types/types.go` | Pod, Node, Container, Status structs | ~180  |

### Example Manifests (`manifests/`)

```yaml
manifests/
├── example-pod.yaml           # Basic nginx pod
├── multi-container-pod.yaml   # Pod with multiple containers
├── redis-pod.yaml             # Redis database pod
└── busybox-pod.yaml           # Debug/testing pod
```

### Scripts (`scripts/`)

```bash
scripts/
├── start-all.sh              # Starts API server, scheduler, kubelet
├── stop-all.sh               # Gracefully stops all components
├── build.sh                  # Builds all binaries
├── test-cluster.sh           # Runs integration tests
└── cleanup.sh                # Removes containers and resets state
```

## Creating the Complete Structure

Run these commands to create the folder structure:

```bash
# Create main directories
mkdir -p cmd/{apiserver,scheduler,kubelet,minictl}
mkdir -p pkg/{api,storage,scheduler,kubelet,controller}
mkdir -p types
mkdir -p manifests
mkdir -p bin
mkdir -p scripts
mkdir -p docs
mkdir -p configs
mkdir -p test/{integration,unit}

# Create placeholder files
touch cmd/apiserver/main.go
touch cmd/scheduler/main.go
touch cmd/kubelet/main.go
touch cmd/minictl/main.go

touch pkg/api/handlers.go
touch pkg/storage/store.go
touch pkg/scheduler/scheduler.go
touch pkg/kubelet/kubelet.go
touch pkg/controller/controller.go

touch types/types.go

touch manifests/example-pod.yaml
touch manifests/multi-container-pod.yaml

touch README.md
touch LICENSE
touch Makefile
touch .gitignore

echo "Folder structure created successfully!"
```

## Additional Files You Might Want

### Makefile

```makefile
.PHONY: build clean test run-api run-scheduler run-kubelet

build:
	@echo "Building all components..."
	go build -o bin/apiserver cmd/apiserver/main.go
	go build -o bin/scheduler cmd/scheduler/main.go
	go build -o bin/kubelet cmd/kubelet/main.go
	go build -o bin/minictl cmd/minictl/main.go

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

test:
	go test ./...

run-api:
	./bin/apiserver

run-scheduler:
	./bin/scheduler

run-kubelet:
	./bin/kubelet --node-name=node1
```

### .gitignore

```
# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test files
*.test
*.out

# Go workspace file
go.work

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
```

### scripts/start-all.sh

```bash
#!/bin/bash

echo "Starting Mini Kubernetes Cluster..."

# Start API Server
echo "Starting API Server..."
./bin/apiserver &
API_PID=$!
sleep 2

# Start Scheduler
echo "Starting Scheduler..."
./bin/scheduler &
SCHED_PID=$!
sleep 1

# Start Kubelet
echo "Starting Kubelet..."
./bin/kubelet --node-name=node1 &
KUBELET_PID=$!

echo "Cluster started!"
echo "API Server PID: $API_PID"
echo "Scheduler PID: $SCHED_PID"
echo "Kubelet PID: $KUBELET_PID"

echo "Saving PIDs to .pids file..."
echo "$API_PID $SCHED_PID $KUBELET_PID" > .pids
```

### scripts/stop-all.sh

```bash
#!/bin/bash

if [ -f .pids ]; then
    echo "Stopping all components..."
    read API_PID SCHED_PID KUBELET_PID < .pids

    kill $API_PID $SCHED_PID $KUBELET_PID 2>/dev/null

    rm .pids
    echo "All components stopped!"
else
    echo "No running components found."
fi
```

## File Size Estimates

| Component    | Files  | Total Lines | Size (approx) |
| ------------ | ------ | ----------- | ------------- |
| Core Types   | 1      | 180         | 5 KB          |
| Storage      | 1      | 180         | 6 KB          |
| API Handlers | 1      | 250         | 8 KB          |
| Scheduler    | 1      | 200         | 7 KB          |
| Kubelet      | 1      | 250         | 9 KB          |
| Controller   | 1      | 150         | 5 KB          |
| CLI Tool     | 1      | 400         | 14 KB         |
| Main Apps    | 4      | 180         | 6 KB          |
| **Total**    | **11** | **~1,790**  | **~60 KB**    |

## Directory Purpose Summary

- **`cmd/`** - Executable entry points for each component
- **`pkg/`** - Reusable libraries and business logic
- **`types/`** - Shared data structures
- **`manifests/`** - Example pod configurations
- **`bin/`** - Compiled binaries (git-ignored)
- **`scripts/`** - Automation and helper scripts
- **`docs/`** - Documentation files
- **`configs/`** - Configuration files
- **`test/`** - Test files (unit and integration)

This structure follows Go best practices and makes the codebase maintainable and scalable!
