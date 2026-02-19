package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestMain initializes the database before running tests
func TestMain(m *testing.M) {
	// Initialize database
	initDB()

	// Run tests
	code := m.Run()

	// Cleanup
	if DB != nil {
		DB.Close()
	}

	os.Exit(code)
}

// Helper function to create a test router with the handler
func createTestRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/{table}", HandleSelect)
	return r
}

// TestLeftJoinBasic tests basic left join (default behavior)
func TestLeftJoinBasic(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts(id,content)", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) == 0 {
		t.Log("No results returned (expected if no authors exist)")
	} else {
		t.Logf("Left join basic test passed. Got %d authors", len(result))
		t.Logf("Response sample: %v", result[0])
	}
}

// TestInnerJoinBasic tests basic inner join with !inner syntax
func TestInnerJoinBasic(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts!inner(id,content)", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) == 0 {
		t.Log("No results returned - inner join filters authors without posts")
	} else {
		t.Logf("Inner join basic test passed. Got %d authors with posts", len(result))
		t.Logf("Response sample: %v", result[0])
	}
}

// TestMixedJoins tests mixed join types (inner join for posts, left join for stats)
func TestMixedJoins(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts!inner(id,content,stats(id,views))", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) == 0 {
		t.Log("No results returned - inner join on posts filters authors")
	} else {
		t.Logf("Mixed joins test passed. Got %d authors", len(result))
		if len(result) > 0 {
			t.Logf("Response sample: %v", result[0])
		}
	}
}

// TestAllInnerJoins tests all inner joins at multiple nesting levels
func TestAllInnerJoins(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) == 0 {
		t.Log("No results returned - both inner joins filter results")
	} else {
		t.Logf("All inner joins test passed. Got %d authors", len(result))
		if len(result) > 0 {
			t.Logf("Response sample: %v", result[0])
		}
	}
}

// TestNestedLeftJoins tests nested left joins (multiple levels without inner)
func TestNestedLeftJoins(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts(id,content,stats(id,views))", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) == 0 {
		t.Log("No results returned (expected if no authors exist)")
	} else {
		t.Logf("Nested left joins test passed. Got %d authors", len(result))
		t.Logf("Response sample: %v", result[0])
	}
}

// TestMissingTenantHeader tests that missing X-Tenant-ID header is rejected
func TestMissingTenantHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name", nil)
	// Intentionally not setting X-Tenant-ID header

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Missing X-Tenant-ID header")) {
		t.Errorf("Expected error message about missing header, got: %s", w.Body.String())
	}
}

// TestSimpleSelect tests basic select without joins
func TestSimpleSelect(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Simple select test passed. Got %d authors", len(result))
}

// TestSelectAllColumns tests select with wildcard
func TestSelectAllColumns(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Select all columns test passed. Got %d authors", len(result))
}

// TestInnerJoinOnlyStats tests inner join on nested level only
func TestInnerJoinOnlyStats(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts(id,content,stats!inner(id,views))", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Inner join on stats only test passed. Got %d authors", len(result))
	if len(result) > 0 {
		t.Logf("Response sample: %v", result[0])
	}
}

// TestWithoutSelect tests query without select parameter
func TestWithoutSelect(t *testing.T) {
	req := httptest.NewRequest("GET", "/authors", nil)
	req.Header.Set("X-Tenant-ID", "public")

	w := httptest.NewRecorder()
	router := createTestRouter()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Error: %s", w.Code, w.Body.String())
	}

	var result []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Without select parameter test passed. Got %d authors", len(result))
}

// IntegrationTestAllEndpoints runs all tests and prints summary
func TestIntegrationAllEndpoints(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldPass  bool
		description string
	}{
		{
			name:        "SimpleSelect",
			url:         "/authors?select=id,first_name",
			shouldPass:  true,
			description: "Simple select without joins (LEFT JOIN default)",
		},
		{
			name:        "LeftJoinBasic",
			url:         "/authors?select=id,first_name,posts(id,content)",
			shouldPass:  true,
			description: "Basic LEFT JOIN on posts",
		},
		{
			name:        "InnerJoinBasic",
			url:         "/authors?select=id,first_name,posts!inner(id,content)",
			shouldPass:  true,
			description: "Basic INNER JOIN on posts (filters authors without posts)",
		},
		{
			name:        "NestedLeftJoins",
			url:         "/authors?select=id,first_name,posts(id,content,stats(id,views))",
			shouldPass:  true,
			description: "Nested LEFT JOINs on posts and stats",
		},
		{
			name:        "MixedJoins",
			url:         "/authors?select=id,first_name,posts!inner(id,content,stats(id,views))",
			shouldPass:  true,
			description: "INNER JOIN on posts, LEFT JOIN on stats",
		},
		{
			name:        "AllInnerJoins",
			url:         "/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))",
			shouldPass:  true,
			description: "INNER JOIN on posts AND stats (most restrictive)",
		},
		{
			name:        "InnerJoinOnlyStats",
			url:         "/authors?select=id,first_name,posts(id,content,stats!inner(id,views))",
			shouldPass:  true,
			description: "LEFT JOIN on posts, INNER JOIN on stats",
		},
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("INTEGRATION TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	router := createTestRouter()
	passed := 0
	failed := 0

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.url, nil)
		req.Header.Set("X-Tenant-ID", "public")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		success := w.Code == http.StatusOK
		if success == test.shouldPass {
			passed++
			status := "✓ PASS"
			fmt.Printf("%s | %s\n", status, test.name)
		} else {
			failed++
			status := "✗ FAIL"
			fmt.Printf("%s | %s\n", status, test.name)
			fmt.Printf("     Error: %s\n", w.Body.String())
		}

		fmt.Printf("     Description: %s\n", test.description)
		fmt.Printf("     URL: %s\n", test.url)
		fmt.Printf("     Status Code: %d\n\n", w.Code)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Total Passed: %d/%d\n", passed, passed+failed)
	fmt.Printf("Total Failed: %d/%d\n", failed, passed+failed)
	fmt.Println(strings.Repeat("=", 80) + "\n")

	if failed > 0 {
		t.Errorf("Integration test failed: %d/%d tests failed", failed, passed+failed)
	}
}

// Benchmark tests for performance monitoring
func BenchmarkInnerJoin(b *testing.B) {
	router := createTestRouter()
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts!inner(id,content)", nil)
	req.Header.Set("X-Tenant-ID", "public")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkLeftJoin(b *testing.B) {
	router := createTestRouter()
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts(id,content)", nil)
	req.Header.Set("X-Tenant-ID", "public")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkNestedInnerJoins(b *testing.B) {
	router := createTestRouter()
	req := httptest.NewRequest("GET", "/authors?select=id,first_name,posts!inner(id,content,stats!inner(id,views))", nil)
	req.Header.Set("X-Tenant-ID", "public")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
