all: run

fmt:
	go fmt

run:
	go build
	./httptester simple.htc

cover:
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out
