.DEFAULT_GOAL := test

setup:
	dep ensure

fmt:
	go fmt

get:
	go vet

test:
	go test -race ./...

coverage:
	bash ./.run-coverage.sh
