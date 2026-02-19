# TenantRest API Testing Guide

## Quick Start

### 1. Run the Application

```bash
# Navigate to the project directory
cd /home/maitretech/tenantrest

# Build and run the application
go run main.go
```

The application will start on `http://localhost:8080`

### 2. Run Unit Tests

```bash
# Run all tests
go test -v

# Run with integration test summary
go test -v -run TestIntegration

# Run specific test
go test -v -run TestInnerJoinBasic

# Run benchmarks
go test -bench=. -benchmem

# Run tests with coverage
go test -cover
```

---

## CURL Command Examples

### Important Header
All requests require the `X-Tenant-ID` header:
```bash
-H "X-Tenant-ID: public"
```

---

## Test Scenarios

### 1. Simple Select (No Joins)
**Description**: Fetch authors with specific columns

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name" \
  -H "X-Tenant-ID: public"
```

**What it does**: Returns authors with only `id` and `first_name` columns

---

### 2. Basic Left Join (Default)
**Description**: Fetch authors with their posts using LEFT JOIN (returns all authors, even those without posts)

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts(id,content)" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Returns all authors
- Includes `posts` as a JSON array (empty array if author has no posts)

---

### 3. Basic Inner Join
**Description**: Fetch authors WITH posts using INNER JOIN (only returns authors who have posts)

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts!inner(id,content)" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Only returns authors who have at least one post
- Includes `posts` as a JSON array
- Note the `!inner` syntax after `posts`

---

### 4. Nested Left Joins
**Description**: Fetch authors with their posts and post statistics using LEFT JOIN at all levels

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts(id,content,stats(id,views))" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Returns all authors (LEFT JOIN with posts)
- For each post, includes `stats` array (LEFT JOIN with stats)
- Missing data results in empty arrays

---

### 5. Mixed Join Types
**Description**: INNER JOIN on posts, LEFT JOIN on stats

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts!inner(id,content,stats(id,views))" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Only returns authors who have posts (INNER JOIN)
- For each post, includes `stats` as array (LEFT JOIN)

---

### 6. All Inner Joins
**Description**: INNER JOIN at multiple levels (most restrictive)

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Only returns authors who have posts (INNER JOIN)
- Only includes posts that have stats (INNER JOIN)
- Most filtered results

---

### 7. Inner Join Only on Nested Level
**Description**: LEFT JOIN on posts, INNER JOIN on stats

```bash
curl -X GET "http://localhost:8080/authors?select=id,first_name,posts(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: public"
```

**What it does**:
- Returns all authors (LEFT JOIN with posts)
- For each post, only includes if it has stats (INNER JOIN)

---

### 8. Select All Columns
**Description**: Fetch all columns from authors (default behavior when no select parameter)

```bash
curl -X GET "http://localhost:8080/authors" \
  -H "X-Tenant-ID: public"
```

---

## Testing Script

Run all CURL commands sequentially:

```bash
#!/bin/bash

TENANT_ID="public"
BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Running All Test Cases"
echo "=========================================="

echo -e "\n1. Simple Select (No Joins)"
curl -X GET "$BASE_URL/authors?select=id,first_name" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n2. Basic Left Join"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content)" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n3. Basic Inner Join"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content)" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n4. Nested Left Joins"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content,stats(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n5. Mixed Join Types"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n6. All Inner Joins"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n7. Inner Join Only on Nested Level"
curl -X GET "$BASE_URL/authors?select=id,first_name,posts(id,content,stats!inner(id,views))" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n8. Select All Columns"
curl -X GET "$BASE_URL/authors" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .

echo -e "\n=========================================="
echo "Tests Complete"
echo "=========================================="
```

Save as `test.sh` and run:
```bash
chmod +x test.sh
./test.sh
```

---

## Go Test Commands

### Run All Tests with Verbose Output
```bash
go test -v
```

### Run Specific Test
```bash
go test -v -run TestInnerJoinBasic
```

### Run Integration Test
```bash
go test -v -run IntegrationTestAllEndpoints
```

### Run Benchmarks
```bash
go test -bench=. -benchmem -run=^$
```

### Run with Race Detector
```bash
go test -race
```

### Generate Coverage Report
```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Expected Response Format

All responses are JSON arrays of objects:

```json
[
  {
    "id": 1,
    "first_name": "John",
    "posts": [
      {
        "id": 101,
        "content": "First post",
        "stats": [
          {
            "id": 1001,
            "views": 1500
          }
        ]
      }
    ]
  }
]
```

For INNER JOIN scenarios, if an author has no posts, that author will not appear in the results.

---

## Join Type Comparison

| Scenario | Query | Behavior |
|----------|-------|----------|
| All authors (with/without posts) | `posts(...)` | LEFT JOIN - returns empty array if no posts |
| Authors with posts only | `posts!inner(...)` | INNER JOIN - filters out authors without posts |
| Mixed | `posts!inner(...,stats(...))` | INNER on posts, LEFT on stats |
| Most restrictive | `posts!inner(...,stats!inner(...))` | INNER on both - most filtered |

---

## Troubleshooting

### "Missing X-Tenant-ID header" error
**Solution**: Always include `-H "X-Tenant-ID: public"` in your curl commands

### "table name ... specified more than once" error
**Solution**: This was a bug that has been fixed. Update your code to the latest version.

### "column ... does not exist" error
**Solution**: Check that the column names in the select parameter exist in your database tables

### Getting null or empty results
**Solution**: 
- For INNER JOINs: Check that related data actually exists
- For LEFT JOINs: Empty arrays are normal when no related rows exist

