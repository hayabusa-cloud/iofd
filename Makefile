.PHONY: all test bench vet clean

all: vet test

test:
	go test -timeout 120s -race ./...

bench:
	go test -timeout 120s -bench=. -benchmem -run=^$$ ./...

vet:
	go vet ./...

clean:
	rm -f coverage.out
