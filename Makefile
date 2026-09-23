lint:
	golangci-lint run
test:
	go test ./...
test-v:
	go test ./... -v
build-image:
	docker build -t sonarkube .