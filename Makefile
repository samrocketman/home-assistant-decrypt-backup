.PHONY: lint release release-snapshot test ci upgrade-actions clean

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

ci:
	docker build -f Dockerfile.test -t hassio-tar-test .
	mkdir -p test-results
	docker run --rm hassio-tar-test goss -g tests/goss.yaml validate --format junit > test-results/goss-report.xml

upgrade-actions:
	bash scripts/upgrade-actions.sh

clean:
	rm -rf dist/ release/ test-results/ hassio-tar
