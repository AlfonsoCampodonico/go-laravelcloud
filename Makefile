.PHONY: run build test fmt vet tidy docker clean

run:
	go run .

build:
	go build -o bin/api .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

docker:
	docker build -t go-laravelcloud:dev .

clean:
	rm -rf bin
