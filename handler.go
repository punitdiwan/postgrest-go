package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func HandleSelect(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "table")
	log.Println("Table requested:", table)
	tenants := r.Header.Get("X-Tenant-ID")
	log.Println("Tenant ID:", tenants)
	if tenants == "" {
		http.Error(w, "Missing X-Tenant-ID header", http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	tx, err := DB.Begin(ctx)
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Tenant isolation: ensure queries are scoped to the tenant
	log.Println("Setting the Saearch Path")
	_, err = tx.Exec(ctx, fmt.Sprintf(`SET LOCAL search_path TO "%s"`, tenants))
	if err != nil {
		http.Error(w, "Failed to set tenant context", http.StatusInternalServerError)
		return
	}

	sql := BuildQuery(ctx, tx, table, r.URL.Query())
	log.Println("SQL Query is:", sql.Query)
	log.Println("SQL Values are:", sql.Values)
	rows, err := tx.Query(ctx, sql.Query, sql.Values...)
	if err != nil {
		http.Error(w, "Query execution failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Process rows and write response.
	fields := rows.FieldDescriptions()
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(fields))
		valuePtrs := make([]interface{}, len(fields))
		for i := range fields {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}

		rowMap := make(map[string]interface{})
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}

		results = append(results, rowMap)
	}
	tx.Commit(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
