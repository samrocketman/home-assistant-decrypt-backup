FROM ubuntu:22.04
SHELL ["/bin/bash", "-euxo", "pipefail", "-c"]
RUN \
  apt-get update; \
  apt-get install -y curl vim git make xz-utils gcc; \
  curl -sSfLo /usr/local/bin/download-utilities.sh https://raw.githubusercontent.com/samrocketman/yml-install-files/33f873043002ef1923859046652b284581b988b6/download-utilities.sh; \
  chmod 755 /usr/local/bin/download-utilities.sh; \
  curl -sSfL https://raw.githubusercontent.com/samrocketman/goenv/c742492b3a44a094a86a53eefa8efa6060c771ab/go.yml | \
    download-utilities.sh - go=1.23.12 tinygo=0.40.1 upx=5.1.0

# prepare a base scratch
RUN \
  mkdir -p base/bin base/tmp base/var/tmp base/etc base/home/nonroot base/sbin base/root; \
  chmod 700 base/root; \
  chown root:root base/root; \
  chmod 1777 base/tmp base/var/tmp; \
  chown 65532:65532 base/home/nonroot; \
  chmod 750 base/home/nonroot; \
  echo 'root:x:0:' > /base/etc/group; \
  echo 'nonroot:x:65532:' >> /base/etc/group; \
  echo 'root:x:0:0:root:/root:/sbin/nologin' > /base/etc/passwd; \
  echo 'nonroot:x:65532:65532:nonroot:/home/nonroot:/sbin/nologin' >> /base/etc/passwd; \
  echo 'int main() { return 1; }' > nologin.c; \
  gcc -Os -s -o /base/sbin/nologin nologin.c; \
  strip --strip-all /base/sbin/nologin


ENV PATH=/usr/local/go/bin:/usr/local/tinygo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
WORKDIR /opt
COPY go.mod go.sum hassio-tar.go secretstream.go ./
ENV CGO_ENABLED=0
RUN \
  tinygo build -o hassio-tar .; \
  upx hassio-tar; \
  mv hassio-tar /base/bin/

FROM scratch
COPY --from=0 /base/ /
ENTRYPOINT ["/bin/hassio-tar"]
USER nonroot
ENV HOME=/home/nonroot USER=nonroot
WORKDIR /home/nonroot
