#!/bin/bash

# test-routing.sh - Test script for cell routing system
# Usage: ./test-routing.sh

set -e

ROUTER_URL="${ROUTER_URL:-http://localhost:8080}"
ENDPOINT="/api/test"

echo "🧪 Testing Cell Routing System"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_routing() {
    local name="$1"
    local routing_key="$2"
    local expected_placement="$3"
    local expected_reason="$4"
    
    echo -e "${BLUE}Test: ${name}${NC}"
    echo "  Routing Key: ${routing_key:-<none>}"
    
    if [ -z "$routing_key" ]; then
        response=$(curl -s -i "${ROUTER_URL}${ENDPOINT}" 2>&1)
    else
        response=$(curl -s -i -H "X-Routing-Key: ${routing_key}" "${ROUTER_URL}${ENDPOINT}" 2>&1)
    fi
    
    routed_to=$(echo "$response" | grep -i "X-Routed-To:" | cut -d' ' -f2 | tr -d '\r')
    route_reason=$(echo "$response" | grep -i "X-Route-Reason:" | cut -d' ' -f2 | tr -d '\r')
    status_code=$(echo "$response" | grep -i "HTTP/" | head -1 | awk '{print $2}')
    
    echo "  Status: ${status_code}"
    echo "  X-Routed-To: ${routed_to}"
    echo "  X-Route-Reason: ${route_reason}"
    
    if [ "$routed_to" = "$expected_placement" ] && [ "$route_reason" = "$expected_reason" ]; then
        echo -e "  ${GREEN}✓ PASS${NC}"
    else
        echo -e "  ${YELLOW}✗ FAIL${NC} (expected: placement=${expected_placement}, reason=${expected_reason})"
    fi
    echo ""
}

# Wait for router to be ready
echo "Waiting for router to be ready..."
for i in {1..30}; do
    if curl -s -o /dev/null -w "%{http_code}" "${ROUTER_URL}/health" 2>/dev/null; then
        echo "Router is ready!"
        echo ""
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Router failed to start. Make sure 'docker compose up' is running."
        exit 1
    fi
    sleep 1
done

# Run tests
test_routing "Dedicated customer (visa)" "visa" "visa" "dedicated"
test_routing "Tier 1 customer (acme)" "acme" "tier1" "tier"
test_routing "Tier 2 customer (globex)" "globex" "tier2" "tier"
test_routing "Tier 3 customer (initech)" "initech" "tier3" "tier"
test_routing "Unknown routing key" "unknown-company" "tier3" "default"
test_routing "Missing routing key" "" "tier3" "default"

echo "================================"
echo -e "${GREEN}✓ All routing tests completed${NC}"
echo ""
echo "💡 Tip: Check router logs with: docker compose logs -f router"
