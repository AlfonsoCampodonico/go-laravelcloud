.PHONY: run build test fmt vet tidy docker clean smoke

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

smoke:
	@PORT=9114 go run . & echo $$! > /tmp/golc.pid; sleep 1; \
	curl -s -o /dev/null -w "GET /openapi.json -> %{http_code}\n" localhost:9114/openapi.json; \
	curl -s -o /dev/null -w "GET /docs/ -> %{http_code}\n" localhost:9114/docs/; \
	kill `cat /tmp/golc.pid`
