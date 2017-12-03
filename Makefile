
test:
	go test ./... -v -race -cover

bench:
	go test --bench=. --benchmem

.PHONY: test
