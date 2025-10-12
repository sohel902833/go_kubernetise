.PHONY: dev build start clean

dev:
	@echo "💻 Running in development mode (hot reload with Air)..."
	air

build:
	@echo "🏗️ Building production binary..."
	@mkdir -p bin
	go build -o bin/app ./src/main.go

start:
	@echo "🚀 Starting production server..."
	./bin/app

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin tmp
