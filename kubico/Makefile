.PHONY: all build clean test run-api run-scheduler run-kubelet run-all stop help

# Variables
BINARY_DIR=bin
API_BINARY=$(BINARY_DIR)/apiserver
SCHEDULER_BINARY=$(BINARY_DIR)/scheduler
KUBELET_BINARY=$(BINARY_DIR)/kubelet
CONTROLLER_BINARY=$(BINARY_DIR)/controller
CLI_BINARY=$(BINARY_DIR)/minictl

# Go parameters
GOCMD=/usr/local/go/bin/go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build all binaries
all: clean build

# Build all components
build: build-api build-scheduler build-kubelet build-cli build-controller
	@echo "✅ All components built successfully!"

# Build API Server
build-api:
	@echo "🔨 Building API Server..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(API_BINARY) cmd/apiserver/main.go
#Build Controller
build-controller:
	@echo "🔨 Building Controller..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(CONTROLLER_BINARY) cmd/controller/main.go

# Build Scheduler
build-scheduler:
	@echo "🔨 Building Scheduler..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(SCHEDULER_BINARY) cmd/scheduler/main.go

# Build Kubelet
build-kubelet:
	@echo "🔨 Building Kubelet..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(KUBELET_BINARY) cmd/kubelet/main.go

# Build CLI
build-cli:
	@echo "🔨 Building CLI (minictl)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(CLI_BINARY) cmd/minictl/main.go

# Run API Server
run-api: build-api
	@echo "🚀 Starting API Server..."
	$(API_BINARY) --port 8080

# Run Controller
run-controller: build-controller
	@echo "🚀 Starting Controller..."
	$(CONTROLLER_BINARY) --api-server http://localhost:8080

# Run Scheduler
run-scheduler: build-scheduler
	@echo "🚀 Starting Scheduler..."
	$(SCHEDULER_BINARY) --api-server http://localhost:8080

# Run Kubelet
run-kubelet: build-kubelet
	@echo "🚀 Starting Kubelet..."
	$(KUBELET_BINARY) --node-name node1 --api-server http://localhost:8080

# Run all components (requires tmux or separate terminals)
run-all: build
	@echo "🚀 Starting all components..."
	@echo "Note: Run each component in separate terminals:"
	@echo "  Terminal 1: make run-api"
	@echo "  Terminal 2: make run-scheduler"
	@echo "  Terminal 3: make run-kubelet"

# Install dependencies
deps:
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "💅 Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "🔍 Linting code..."
	golangci-lint run

# Create necessary directories
init:
	@echo "📁 Creating project structure..."
	@mkdir -p cmd/{apiserver,scheduler,kubelet,minictl}
	@mkdir -p pkg/{api,storage,scheduler,kubelet,controller}
	@mkdir -p types manifests bin scripts docs configs test/{integration,unit}
	@echo "✅ Project structure created!"

# Install CLI globally
install-cli: build-cli
	@echo "📦 Installing minictl..."
	cp $(CLI_BINARY) /usr/local/bin/minictl
	@echo "✅ minictl installed to /usr/local/bin/minictl"
#manual installation
# make build-cli
# sudo cp bin/minictl /usr/local/bin/minictl

# Uninstall CLI
uninstall-cli:
	@echo "🗑️  Uninstalling minictl..."
	rm -f /usr/local/bin/minictl
	@echo "✅ minictl uninstalled"

# Docker: Build image (for future containerization)
docker-build:
	@echo "🐳 Building Docker images..."
	docker build -t mini-k8s-apiserver:latest -f docker/Dockerfile.apiserver .
	docker build -t mini-k8s-scheduler:latest -f docker/Dockerfile.scheduler .
	docker build -t mini-k8s-kubelet:latest -f docker/Dockerfile.kubelet .

# Show help
help:
	@echo "Mini Kubernetes Clone - Makefile Commands"
	@echo ""
	@echo "Building:"
	@echo "  make build           - Build all components"
	@echo "  make build-api       - Build API Server only"
	@echo "  make build-scheduler - Build Scheduler only"
	@echo "  make build-kubelet   - Build Kubelet only"
	@echo "  make build-cli       - Build CLI only"
	@echo ""
	@echo "Running:"
	@echo "  make run-api         - Run API Server"
	@echo "  make run-scheduler   - Run Scheduler"
	@echo "  make run-kubelet     - Run Kubelet"
	@echo "  make run-all         - Show instructions to run all"
	@echo ""
	@echo "Testing:"
	@echo "  make test            - Run tests"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make deps            - Download dependencies"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo ""
	@echo "Installation:"
	@echo "  make install-cli     - Install minictl globally"
	@echo "  make uninstall-cli   - Uninstall minictl"
	@echo ""
	@echo "Setup:"
	@echo "  make init            - Create project structure"