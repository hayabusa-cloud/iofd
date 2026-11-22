BENCH ?= .

.PHONY: all test bench vet clean

all: vet test

test:
	go test -timeout 120s -race ./...

bench:
	go test -run='^$$' -bench='$(BENCH)' -benchmem -benchtime=200ms -count=10 -cpu=1 -timeout=5m ./...

vet:
	go vet ./...

clean:
	rm -f coverage.out
