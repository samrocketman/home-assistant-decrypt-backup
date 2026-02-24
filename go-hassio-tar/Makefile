.PHONY: release
release:
	@if [ -d release ]; then echo run make clean first; exit 1; fi
	docker build -t htar-release -f Dockerfile.release .
	docker run htar-release cat /release.tar | tar -x
clean:
	rm -rf release
