.PHONY: all test bench vet clean

all: vet test

test:
	go test -race ./...

bench:
	go test -bench=. -benchmem -run=^$$ ./...

vet:
	go vet ./...

clean:
	rm -f coverage.out
