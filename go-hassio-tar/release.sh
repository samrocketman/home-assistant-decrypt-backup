#!/bin/bash
# Created by Sam Gleske
# Pop!_OS 22.04 LTS
# Linux 6.9.3-76060903-generic x86_64
# GNU bash, version 5.1.16(1)-release (x86_64-pc-linux-gnu)
#
# builds binaries:
#  - linux (armv6, armv7, aarch64, x86_64)
#  - mac (arm64 and x86_64)
#  - windows (arm64 and x86_64)
set -exuo pipefail
mkdir -p release;
for os in linux darwin windows; do
  for arch in 386 amd64 arm64 arm; do
    if {
        case "${os}-${arch}" in
          darwin-386|darwin-arm|windows-386|windows-arm) true;;
          *) false;;
        esac
      }; then
      continue;
    fi;
    if [ "${os}" = windows ]; then
      ext=".exe";
    else
      ext="";
    fi;
    export GOARCH="${arch}" GOOS="${os}";
    if [ "${arch}" = arm ]; then
      GOARM=6 tinygo build -o "release/hassio-tar-${os}-armv6" hassio-tar.go;
      upx "release/hassio-tar-${os}-armv6"
      GOARM=7 tinygo build -o "release/hassio-tar-${os}-armv7" hassio-tar.go;
      upx "release/hassio-tar-${os}-armv7"
    else
      tinygo build -o "release/hassio-tar-${os}-${arch/386/i386}${ext:-}" hassio-tar.go;
      if [ "$os" = linux ] || {
          [ "$os" = windows ] && [ "$arch" = amd64 ]
        }; then
        upx "release/hassio-tar-${os}-${arch/386/i386}${ext:-}"
      fi
    fi;
  done;
done;
tar -cf /release.tar release
