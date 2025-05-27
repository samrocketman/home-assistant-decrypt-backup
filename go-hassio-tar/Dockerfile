FROM alpine
SHELL ["/bin/sh", "-exc"]
RUN \
  apk add bash curl git; \
  curl -sSfLo /usr/local/bin/download-utilities.sh https://raw.githubusercontent.com/samrocketman/yml-install-files/33f873043002ef1923859046652b284581b988b6/download-utilities.sh; \
  chmod 755 /usr/local/bin/download-utilities.sh; \
  curl -sSfL https://raw.githubusercontent.com/samrocketman/goenv/c2263e6e72fa65600db203db95a315f99b6dfbd7/go.yml | \
    download-utilities.sh - go tinygo

# prepare a base scratch
RUN \
  apk add --no-cache build-base; \
  mkdir -p base/bin base/tmp base/var/tmp base/etc base/home/nonroot base/sbin base/root; \
  chmod 700 /root; \
  chown root:root /root; \
  chmod 1777 base/tmp base/var/tmp; \
  chown 65532:65532 base/home/nonroot; \
  chmod 750 base/home/nonroot; \
  echo 'root:x:0:' > /base/etc/group; \
  echo 'nonroot:x:65532:' >> /base/etc/group; \
  echo 'root:x:0:0:root:/root:/sbin/nologin' > /base/etc/passwd; \
  echo 'nonroot:x:65532:65532:nonroot:/home/nonroot:/sbin/nologin' >> /base/etc/passwd; \
  echo 'int main() { return 1; }' > nologin.c; \
  gcc -Os -no-pie -static -std=gnu99 -s -Wall -Werror -o /base/sbin/nologin nologin.c

ENV PATH=/usr/local/go/bin:/usr/local/tinygo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
COPY hassio-tar.go hassio-tar.go
RUN  tinygo build hassio-tar.go; \
  mv hassio-tar /base/bin/

FROM scratch
COPY --from=0 /base/ /
ENTRYPOINT ["/bin/hassio-tar"]
USER nonroot
ENV HOME=/home/nonroot USER=nonroot
WORKDIR /home/nonroot
