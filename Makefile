.PHONY: help dev build test run clean build-backend build-frontend test-backend test-frontend run-backend run-frontend clean-backend clean-frontend install

# Default target
help:
	@echo "Available targets:"
	@echo "  make dev              - Install all dependencies (backend + frontend)"
	@echo "  make build            - Build both backend and frontend"
	@echo "  make build-backend    - Build Go backend binary"
	@echo "  make build-frontend   - Build frontend production bundle"
	@echo "  make test             - Run all tests (backend + frontend)"
	@echo "  make test-backend     - Run Go backend tests"
	@echo "  make test-frontend    - Run frontend tests (if configured)"
	@echo "  make run              - Run both backend and frontend (in parallel)"
	@echo "  make run-backend      - Run Go backend server"
	@echo "  make run-frontend     - Run frontend dev server"
	@echo "  make clean            - Clean all build artifacts"
	@echo "  make clean-backend    - Clean Go build artifacts"
	@echo "  make clean-frontend   - Clean frontend build artifacts"
	@echo "  make install          - Alias for 'dev' (install dependencies)"

# Install dependencies
dev install:
	@echo "Setting up development environment..."
	@echo "Installing Go dependencies..."
	cd backend && go mod tidy
	@echo "Installing npm dependencies..."
	cd frontend && npm install
	@echo "Development environment ready!"

# Build targets
build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	cd backend && go build -o server ./cmd/server
	@echo "Backend built: backend/server"

build-frontend:
	@echo "Building frontend..."
	cd frontend && npm run build
	@echo "Frontend built: frontend/dist"

# Test targets
test: test-backend

test-backend:
	@echo "Running backend tests..."
	cd backend && go test ./...

test-frontend:
	@echo "Frontend tests not yet configured"
	@echo "To add tests, configure Vitest or another test framework in frontend/package.json"

# Run targets
run:
	@echo "Starting backend and frontend servers..."
	@echo "Backend will run on http://localhost:8080"
	@echo "Frontend will run on http://localhost:5173"
	@echo "Press Ctrl+C to stop both servers"
	@trap 'kill 0' EXIT; \
	cd backend && go run ./cmd/server & \
	cd frontend && npm run dev & \
	wait

run-backend:
	@echo "Starting backend server..."
	cd backend && go run ./cmd/server

run-frontend:
	@echo "Starting frontend dev server..."
	cd frontend && npm run dev

# Clean targets
clean: clean-backend clean-frontend

clean-backend:
	@echo "Cleaning backend artifacts..."
	cd backend && rm -f server
	cd backend && go clean -cache -testcache
	@echo "Backend cleaned"

clean-frontend:
	@echo "Cleaning frontend artifacts..."
	cd frontend && rm -rf dist node_modules/.vite
	@echo "Frontend cleaned"
