#
# simple makefile to run and build things
#
PROJECT=github.com/pdecat/go-statsd-proxy

.phony: test benchmark format

test:
	@go test -v ${PROJECT}/statsdproxy

benchmark:
	@echo "Running tests..."
	@go test -bench=. ${PROJECT}/statsdproxy

format:
	@go fmt ./statsdproxy
