.PHONY: help run test test-all test-verbose test-coverage test-integration test-benchmarks curl-tests clean build

# Variables
APP_NAME=tenantrest
GO=go
SHELL=/bin/bash

help:
	@echo "TenantRest - Testing Commands"
	@echo "=============================="
	@echo ""
	@echo "Application:"
	@echo "  make run               - Run the application"
	@echo "  make build             - Build the application"
	@echo "  make clean             - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  make test              - Run all unit tests"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-benchmarks   - Run benchmark tests"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-all          - Run all tests (unit + integration + benchmarks)"
	@echo ""
	@echo "CURL Testing:"
	@echo "  make curl-tests        - Run all CURL test scenarios"
	@echo ""
	@echo "Examples:"
	@echo "  make run               - Start the server"
	@echo "  make test-verbose      - Run tests with detailed output"

run:
	@echo "Starting TenantRest application..."
	$(GO) run main.go

build:
	@echo "Building TenantRest..."
	$(GO) build -o $(APP_NAME)

clean:
	@echo "Cleaning build artifacts..."
	$(GO) clean
	rm -f $(APP_NAME)
	rm -f coverage.out coverage.html

test:
	@echo "Running unit tests..."
	$(GO) test -v

test-verbose:
	@echo "Running tests with verbose output..."
	$(GO) test -v -race

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -run IntegrationTestAllEndpoints

test-benchmarks:
	@echo "Running benchmark tests..."
	$(GO) test -bench=. -benchmem -run=^$

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out
	@echo ""
	@echo "Coverage summary:"
	$(GO) tool cover -func=coverage.out | grep total
	@echo ""
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser to view detailed coverage"

test-all: test test-integration test-benchmarks
	@echo ""
	@echo "All tests completed!"

curl-tests:
	@echo "Running CURL test scenarios..."
	@echo "Make sure the application is running (make run in another terminal)"
	@echo ""
	bash test.sh

.DEFAULT_GOAL := help
