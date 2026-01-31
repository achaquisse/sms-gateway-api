.PHONY: help build run test docker-up docker-down docker-logs db-shell clean

help:
	@echo "Available commands:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make docker-up    - Start Docker containers (database + app)"
	@echo "  make docker-down  - Stop Docker containers"
	@echo "  make docker-logs  - View Docker logs"
	@echo "  make db-shell     - Connect to PostgreSQL shell"
	@echo "  make db-only      - Start only the database container"
	@echo "  make clean        - Clean build artifacts"

build:
	cd src && go build -o ../bin/sms-gateway-api .

run:
	cd src && go run .

test:
	cd src && go test -v ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

db-only:
	docker-compose up -d db

db-shell:
	docker exec -it sms-gateway-db psql -U postgres -d sms_gateway

clean:
	rm -rf bin/
	docker-compose down -v
