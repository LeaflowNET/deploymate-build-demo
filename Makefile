.PHONY: build check check-backend check-frontend run
build:
	cd web && npm ci --no-audit --no-fund && npm run build
	go build -o bin/taskmate .
check: check-frontend check-backend
check-frontend:
	cd web && npm run lint && npm test && npm run build
check-backend:
	@test -z "$$(gofmt -l *.go)"
	go vet .
	go test -race .
	go build -o bin/taskmate .
run: build
	./bin/taskmate
