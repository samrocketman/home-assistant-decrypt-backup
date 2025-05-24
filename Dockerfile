FROM alpine
SHELL ["/bin/sh", "-exc"]
COPY hassio-tar.sh /usr/local/bin/
RUN \
  apk add --no-cache \
    bash dumb-init openssl; \
  chmod 755 /usr/local/bin/hassio-tar.sh
ENTRYPOINT ["dumb-init", "--", "/usr/local/bin/hassio-tar.sh"]
