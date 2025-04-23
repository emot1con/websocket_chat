# Build the Go application
build:
	set GOOS=linux&& set GOARCH=amd64&& set CGO_ENABLED=0 && go build -o app/ ./cmd/main.go

# Run the Go application
run: build
	
up_build: build 
	docker-compose down
	docker-compose up --build -d 

# Clean build artifacts
clean:
	rm -f app

# Start Docker containers
up:
	docker-compose up -d

# Stop Docker containers
down:
	docker-compose down

# Run everything
all: docker-up build run

# Stop everything and clean up
stop: docker-down clean
