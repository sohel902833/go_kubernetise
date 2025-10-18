#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PID_FILE=".cluster.pids"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Mini Kubernetes Cluster Shutdown${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

if [ ! -f ${PID_FILE} ]; then
    echo -e "${YELLOW}⚠️  No running cluster found (${PID_FILE} not found)${NC}"
    echo -e "${YELLOW}Checking for running processes...${NC}"
    
    # Try to find and kill any running processes
    pkill -f "bin/apiserver" 2>/dev/null
    pkill -f "bin/scheduler" 2>/dev/null
    pkill -f "bin/kubelet" 2>/dev/null
    
    echo -e "${GREEN}✓ Cleanup complete${NC}"
    exit 0
fi

echo -e "${YELLOW}📋 Reading process IDs...${NC}"

# Read PIDs from file
PIDS=$(cat ${PID_FILE})
PID_ARRAY=($PIDS)

echo -e "${BLUE}Stopping components...${NC}"

# Stop each component
for pid in "${PID_ARRAY[@]}"; do
    if ps -p ${pid} > /dev/null 2>&1; then
        PROCESS_NAME=$(ps -p ${pid} -o comm=)
        echo -e "${YELLOW}  Stopping ${PROCESS_NAME} (PID: ${pid})...${NC}"
        kill ${pid} 2>/dev/null
        
        # Wait up to 5 seconds for graceful shutdown
        for i in {1..5}; do
            if ! ps -p ${pid} > /dev/null 2>&1; then
                echo -e "${GREEN}  ✓ ${PROCESS_NAME} stopped${NC}"
                break
            fi
            sleep 1
            if [ $i -eq 5 ]; then
                echo -e "${RED}  Force killing ${PROCESS_NAME}...${NC}"
                kill -9 ${pid} 2>/dev/null
            fi
        done
    else
        echo -e "${YELLOW}  Process ${pid} not found (already stopped)${NC}"
    fi
done

# Clean up any Docker containers created by the cluster
echo ""
echo -e "${BLUE}🧹 Cleaning up Docker containers...${NC}"
CONTAINERS=$(docker ps -a --filter "label=pod" -q)
if [ ! -z "$CONTAINERS" ]; then
    echo -e "${YELLOW}  Removing pod containers...${NC}"
    docker rm -f $CONTAINERS 2>/dev/null
    echo -e "${GREEN}  ✓ Containers removed${NC}"
else
    echo -e "${GREEN}  ✓ No containers to clean up${NC}"
fi

# Remove PID file
rm ${PID_FILE}

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}✅ Cluster Stopped Successfully!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""