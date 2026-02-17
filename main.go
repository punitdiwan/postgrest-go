package main

import (
	"log"
	"net/http"
)

func main() {
	initDB()
	r := NewRouter()
	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", r)
}
