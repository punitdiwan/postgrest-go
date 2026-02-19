#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TENANT_ID="public"
BASE_URL="http://localhost:8080"

print_test() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    print_error "jq is not installed. Installing jq will make output prettier."
    print_info "Install with: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
    USE_JQ=false
else
    USE_JQ=true
fi

# Check if server is running
print_info "Checking if server is running on $BASE_URL..."
if ! curl -s "$BASE_URL/authors" -H "X-Tenant-ID: $TENANT_ID" > /dev/null 2>&1; then
    print_error "Server is not running on $BASE_URL"
    print_info "Start the server with: go run main.go"
    exit 1
fi
print_success "Server is running"

print_test "TEST 1: Simple Select (No Joins)"
echo "URL: $BASE_URL/authors?select=id,first_name"
echo "Description: Fetch authors with specific columns"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 1 completed"

print_test "TEST 2: Basic Left Join (Default)"
echo "URL: $BASE_URL/authors?select=id,first_name,posts(id,content)"
echo "Description: Fetch authors with their posts using LEFT JOIN"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content)" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 2 completed"

print_test "TEST 3: Basic Inner Join"
echo "URL: $BASE_URL/authors?select=id,first_name,posts!inner(id,content)"
echo "Description: Fetch authors WITH posts using INNER JOIN (only returns authors who have posts)"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content)" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 3 completed"

print_test "TEST 4: Nested Left Joins"
echo "URL: $BASE_URL/authors?select=id,first_name,posts(id,content,stats(id,views))"
echo "Description: Fetch authors with their posts and post statistics using LEFT JOIN at all levels"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content,stats(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 4 completed"

print_test "TEST 5: Mixed Join Types"
echo "URL: $BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats(id,views))"
echo "Description: INNER JOIN on posts, LEFT JOIN on stats"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 5 completed"

print_test "TEST 6: All Inner Joins"
echo "URL: $BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))"
echo "Description: INNER JOIN at multiple levels (most restrictive)"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 6 completed"

print_test "TEST 7: Inner Join Only on Nested Level"
echo "URL: $BASE_URL/authors?select=id,first_name,posts(id,content,stats!inner(id,views))"
echo "Description: LEFT JOIN on posts, INNER JOIN on stats"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 7 completed"

print_test "TEST 8: Select All Columns"
echo "URL: $BASE_URL/authors"
echo "Description: Fetch all columns from authors (default behavior when no select parameter)"
RESPONSE=$(curl -s -X GET "$BASE_URL/authors" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ $USE_JQ = true ]; then
    echo "$RESPONSE" | jq .
else
    echo "$RESPONSE"
fi
print_success "Test 8 completed"

print_test "TESTING SUMMARY"
print_success "All CURL tests completed successfully"
print_info "Run Go unit tests with: go test -v"
print_info "Run Go integration test with: go test -v -run IntegrationTestAllEndpoints"
print_info "Run Go benchmarks with: go test -bench=. -benchmem -run=^$"

echo ""
