
test:
	go test ./... -race -cover

bench:
	go test --bench=. --benchmem

.PHONY: test
