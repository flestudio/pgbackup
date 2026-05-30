default: build

build:
    go build -o bin/pgbackup ./cmd/pgbackup

test:
    go test ./...

test-race:
    go test -race ./...

cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

lint:
    golangci-lint run

fix:
    golangci-lint run --fix

clean:
    rm -rf bin coverage.out

run: build
    ./bin/pgbackup
