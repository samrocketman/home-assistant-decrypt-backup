.PHONY: lint release release-snapshot test clean

lint:
	go fmt ./...
	go vet ./...

release:
	goreleaser release --clean

release-snapshot:
	goreleaser build --snapshot --clean

test:
	docker build -f Dockerfile.test -t hassio-tar-test .
	docker run --rm hassio-tar-test

clean:
	rm -rf dist/ release/ hassio-tar
