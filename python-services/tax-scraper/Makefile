.PHONY: install dev test lint format run proto clean docker

# Python environment
PYTHON := python3
PIP := pip3
VENV := .venv

# Install dependencies
install:
	$(PIP) install -r requirements.txt
	playwright install chromium

# Install with dev dependencies
dev:
	$(PIP) install -e ".[dev]"
	playwright install chromium

# Run tests
test:
	pytest tests/ -v --cov=src --cov-report=term-missing

# Run linting
lint:
	ruff check src/ tests/
	mypy src/

# Format code
format:
	ruff format src/ tests/
	ruff check --fix src/ tests/

# Run the service
run:
	$(PYTHON) -m src.main

# Generate proto code
proto:
	cd ../shared/proto && ./generate.sh

# Clean up
clean:
	rm -rf __pycache__ .pytest_cache .coverage .mypy_cache
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete

# Docker build
docker:
	docker build -t kerp/tax-scraper:latest .

# Docker run
docker-run:
	docker run -p 50051:50051 --env-file .env kerp/tax-scraper:latest

# Help
help:
	@echo "Available targets:"
	@echo "  install    - Install production dependencies"
	@echo "  dev        - Install dev dependencies"
	@echo "  test       - Run tests with coverage"
	@echo "  lint       - Run linting"
	@echo "  format     - Format code"
	@echo "  run        - Run the gRPC service"
	@echo "  proto      - Generate proto code"
	@echo "  clean      - Clean up cache files"
	@echo "  docker     - Build Docker image"
	@echo "  docker-run - Run Docker container"
