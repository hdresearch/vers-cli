#!/bin/bash

# Test script for vers kill command formatting
# This script tests all possible text outputs from the kill command

set -e

echo "🧪 Testing vers kill command formatting..."
echo "============================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build the project
echo -e "${BLUE}Building project...${NC}"
make build

# Function to print section headers
print_section() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

# Function to simulate user input
simulate_input() {
    echo "$1" | ./bin/vers kill "$@"
}

print_section "1. Testing kill command help and usage"
echo "Command: ./bin/vers kill --help"
./bin/vers kill --help || true

print_section "2. Testing invalid arguments"
echo "Command: ./bin/vers kill (no args)"
./bin/vers kill || true

echo -e "\nCommand: ./bin/vers kill -a vm-123 (conflict)"
./bin/vers kill -a vm-123 || true

print_section "3. Creating test environment"
echo "Initializing vers..."
./bin/vers init || true

echo "Creating a test cluster..."
./bin/vers up --cluster-alias test-cluster || {
    echo -e "${RED}Failed to create cluster. Some tests may not work.${NC}"
}

# Wait a moment for cluster to be ready
sleep 2

# Get cluster ID for testing
CLUSTER_ID=$(./bin/vers status --json 2>/dev/null | grep -o '"ClusterID":"[^"]*"' | cut -d'"' -f4 || echo "")
VM_ID=$(./bin/vers status --json 2>/dev/null | grep -o '"ID":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")

if [[ -n "$CLUSTER_ID" ]]; then
    echo -e "${GREEN}Test cluster created: $CLUSTER_ID${NC}"
else
    echo -e "${YELLOW}No cluster created, will test with dummy IDs${NC}"
    CLUSTER_ID="test-cluster-123"
    VM_ID="test-vm-456"
fi

print_section "4. Testing VM deletion scenarios"

echo "Command: ./bin/vers kill nonexistent-vm"
echo "n" | ./bin/vers kill nonexistent-vm || true

echo -e "\nCommand: ./bin/vers kill nonexistent-vm --force"
./bin/vers kill nonexistent-vm --force || true

if [[ -n "$VM_ID" ]]; then
    echo -e "\nCommand: ./bin/vers kill $VM_ID (with cancellation)"
    echo "n" | ./bin/vers kill "$VM_ID" || true
    
    echo -e "\nCommand: ./bin/vers kill $VM_ID (with confirmation, then cancel)"
    echo -e "y\nn" | ./bin/vers kill "$VM_ID" || true
fi

print_section "5. Testing cluster deletion scenarios"

echo "Command: ./bin/vers kill -c nonexistent-cluster"
echo "n" | ./bin/vers kill -c nonexistent-cluster || true

echo -e "\nCommand: ./bin/vers kill -c nonexistent-cluster --force"
./bin/vers kill -c nonexistent-cluster --force || true

if [[ -n "$CLUSTER_ID" ]]; then
    echo -e "\nCommand: ./bin/vers kill -c $CLUSTER_ID (with cancellation)"
    echo "n" | ./bin/vers kill -c "$CLUSTER_ID" || true
fi

print_section "6. Testing kill all clusters scenarios"

echo "Command: ./bin/vers kill -a (with wrong confirmation)"
echo "DELETE SOME" | ./bin/vers kill -a || true

echo -e "\nCommand: ./bin/vers kill -a (with cancellation)"
echo "cancel" | ./bin/vers kill -a || true

echo -e "\nCommand: ./bin/vers kill -a (no clusters case)"
# First delete any existing clusters
if [[ -n "$CLUSTER_ID" ]]; then
    echo "DELETE ALL" | ./bin/vers kill -a || true
fi

# Try kill all when no clusters exist
echo "DELETE ALL" | ./bin/vers kill -a || true

print_section "7. Creating multiple test clusters for bulk operations"

echo "Creating multiple test clusters..."
./bin/vers up --cluster-alias cluster-one || true
sleep 1
./bin/vers up --cluster-alias cluster-two || true
sleep 1
./bin/vers up --cluster-alias cluster-three || true
sleep 2

print_section "8. Testing kill all with multiple clusters"

echo "Command: ./bin/vers kill -a (with correct confirmation)"
echo "DELETE ALL" | ./bin/vers kill -a || true

print_section "9. Testing force operations"

echo "Creating one more cluster for force testing..."
./bin/vers up --cluster-alias force-test-cluster || true
sleep 2

echo -e "\nCommand: ./bin/vers kill -a --force"
./bin/vers kill -a --force || true

print_section "10. Testing edge cases"

echo "Command: ./bin/vers kill '' (empty string)"
echo "n" | ./bin/vers kill '' || true

echo -e "\nCommand: ./bin/vers kill vm-with-very-long-name-that-might-cause-formatting-issues"
echo "n" | ./bin/vers kill vm-with-very-long-name-that-might-cause-formatting-issues || true

echo -e "\nCommand: ./bin/vers kill -c cluster-with-very-long-name-that-might-cause-formatting-issues"
echo "n" | ./bin/vers kill -c cluster-with-very-long-name-that-might-cause-formatting-issues || true

print_section "11. Testing with special characters"

echo "Command: ./bin/vers kill 'vm-with-quotes-and-spaces in name'"
echo "n" | ./bin/vers kill 'vm-with-quotes-and-spaces in name' || true

print_section "12. Testing HEAD impact warnings"

# Create a new cluster and set HEAD
echo "Creating final test cluster..."
./bin/vers up --cluster-alias head-test || true
sleep 2

# Get the current HEAD
HEAD_VM=$(cat .vers/HEAD 2>/dev/null || echo "")
if [[ -n "$HEAD_VM" ]]; then
    echo -e "\nTesting HEAD impact with VM: $HEAD_VM"
    echo "n" | ./bin/vers kill "$HEAD_VM" || true
    
    # Test cluster deletion with HEAD impact
    CURRENT_CLUSTER=$(./bin/vers status --json 2>/dev/null | grep -o '"ClusterID":"[^"]*"' | cut -d'"' -f4 || echo "")
    if [[ -n "$CURRENT_CLUSTER" ]]; then
        echo -e "\nTesting HEAD impact with cluster: $CURRENT_CLUSTER"
        echo "n" | ./bin/vers kill -c "$CURRENT_CLUSTER" || true
    fi
fi

print_section "13. Final cleanup"
echo "Cleaning up test environment..."
echo "DELETE ALL" | ./bin/vers kill -a --force || true

echo -e "\n${GREEN}✅ All formatting tests completed!${NC}"
echo -e "${BLUE}Review the output above to check formatting consistency.${NC}"
echo -e "${YELLOW}Look for:${NC}"
echo "  - Consistent spacing and alignment"
echo "  - Proper emoji and symbol rendering" 
echo "  - Clean line breaks and sections"
echo "  - No weird tabbing or double spacing"
echo "  - Proper color coding (if styles are applied)"
