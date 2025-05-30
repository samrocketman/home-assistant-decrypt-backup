release:
	docker build -t hassio-tar-release -f go-hassio-tar/Dockerfile.release go-hassio-tar
	docker run --rm hassio-tar-release cat /release.tar | tar -x --no-same-owner
	(cd release; for x in *; do sha256sum "$$x" > "$$x".sha256;done)
	docker run --rm hassio-tar-release /bin/bash -c 'go version; tinygo version; upx --version | head -n1' > release/build-environment.txt
clean:
	rm -rf release
