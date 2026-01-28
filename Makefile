.PHONY: dev

dev:
	@echo "Setting up development environment..."
	@echo "Installing Go dependencies..."
	cd backend && go mod tidy
	@echo "Installing npm dependencies..."
	cd frontend && npm install
	@echo "Development environment ready!"
