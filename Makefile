lint:
	golangci-lint run
format:
	gofmt -w .
test:
	go test ./...
test-v:
	go test ./... -v
build-image:
	docker build -t sonarkube .