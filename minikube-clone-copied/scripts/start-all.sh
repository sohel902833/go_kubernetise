#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_PORT=8080
API_SERVER="http://localhost:${API_PORT}"
NODE_NAME="node1"
PID_FILE=".cluster.pids"
LOG_DIR="logs"

# Create logs directory
mkdir -p ${LOG_DIR}

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Mini Kubernetes Cluster Startup${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Check if binaries exist
if [ ! -f "bin/apiserver" ] || [ ! -f "bin/scheduler" ] || [ ! -f "bin/kubelet" ]; then
    echo -e "${YELLOW}⚠️  Binaries not found. Building...${NC}"
    make build
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Build failed!${NC}"
        exit 1
    fi
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running! Please start Docker first.${NC}"
    exit 1
fi

# Check if cluster is already running
if [ -f ${PID_FILE} ]; then
    echo -e "${YELLOW}⚠️  Cluster appears to be running. Stopping existing cluster...${NC}"
    ./scripts/stop-all.sh
    sleep 2
fi

echo -e "${GREEN}✓ Pre-flight checks passed${NC}"
echo ""

# Start API Server
echo -e "${BLUE}🚀 Starting API Server on port ${API_PORT}...${NC}"
./bin/apiserver --port ${API_PORT} > ${LOG_DIR}/apiserver.log 2>&1 &
API_PID=$!
echo ${API_PID} > ${PID_FILE}

# Wait for API server to be ready
echo -e "${YELLOW}⏳ Waiting for API Server to be ready...${NC}"
for i in {1..30}; do
    if curl -s ${API_SERVER}/healthz > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API Server is ready (PID: ${API_PID})${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ API Server failed to start within 30 seconds${NC}"
        cat ${LOG_DIR}/apiserver.log
        kill ${API_PID} 2>/dev/null
        rm ${PID_FILE}
        exit 1
    fi
    sleep 1
    echo -n "."
done
echo ""

# Start Scheduler
echo -e "${BLUE}🚀 Starting Scheduler...${NC}"
./bin/scheduler --api-server ${API_SERVER} > ${LOG_DIR}/scheduler.log 2>&1 &
SCHEDULER_PID=$!
echo ${SCHEDULER_PID} >> ${PID_FILE}
sleep 1
echo -e "${GREEN}✓ Scheduler started (PID: ${SCHEDULER_PID})${NC}"

# Start Kubelet
echo -e "${BLUE}🚀 Starting Kubelet (node: ${NODE_NAME})...${NC}"
./bin/kubelet --node-name ${NODE_NAME} --api-server ${API_SERVER} > ${LOG_DIR}/kubelet.log 2>&1 &
KUBELET_PID=$!
echo ${KUBELET_PID} >> ${PID_FILE}
sleep 2
echo -e "${GREEN}✓ Kubelet started (PID: ${KUBELET_PID})${NC}"

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}✅ Cluster Started Successfully!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo -e "${BLUE}Component Status:${NC}"
echo -e "  API Server:  PID ${API_PID} | Logs: ${LOG_DIR}/apiserver.log"
echo -e "  Scheduler:   PID ${SCHEDULER_PID} | Logs: ${LOG_DIR}/scheduler.log"
echo -e "  Kubelet:     PID ${KUBELET_PID} | Logs: ${LOG_DIR}/kubelet.log"
echo ""
echo -e "${BLUE}API Endpoint:${NC}"
echo -e "  ${API_SERVER}"
echo ""
echo -e "${BLUE}Quick Start:${NC}"
echo -e "  ./bin/minictl get nodes"
echo -e "  ./bin/minictl apply -f manifests/example-pod.yaml"
echo -e "  ./bin/minictl get pods"
echo ""
echo -e "${YELLOW}To stop the cluster: ./scripts/stop-all.sh${NC}"
echo ""

# Verify cluster health
sleep 2
echo -e "${BLUE}🔍 Verifying cluster health...${NC}"
if ./bin/minictl get nodes > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Cluster is healthy!${NC}"
else
    echo -e "${YELLOW}⚠️  Cluster might need a moment to fully initialize${NC}"
fi

echo ""
echo -e "${GREEN}Happy clustering! 🎉${NC}"