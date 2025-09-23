# CNET Agent Makefile

.PHONY: build run test clean install deps

# Build the agent
build:
	go build -o bin/cnet-agent main.go

# Run the agent
run: build
	./bin/cnet-agent -config config.yaml

# Run with debug logging
run-debug: build
	./bin/cnet-agent -config config.yaml

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Install the agent
install: build
	sudo cp bin/cnet-agent /usr/local/bin/
	sudo cp config.yaml /etc/cnet/
	sudo cp scripts/cnet.service /etc/systemd/system/

# Create systemd service
install-service:
	sudo systemctl daemon-reload
	sudo systemctl enable cnet
	sudo systemctl start cnet

# Stop systemd service
stop-service:
	sudo systemctl stop cnet

# View logs
logs:
	journalctl -u cnet -f

# Docker build
docker-build:
	docker build -t cnet-agent .

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml cnet-agent

# Development setup
dev-setup:
	mkdir -p bin
	mkdir -p logs
	go mod tidy
