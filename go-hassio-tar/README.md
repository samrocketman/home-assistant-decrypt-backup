# EXPERIMENTAL hassio-tar - A Go-based version

> This version has been known to fail.  For reliability you should only use the
> shell script `hassio-tar.sh`.


It's a little more minimal than the shell script.  It purely handles streams via
stdin and stdout.  The recommended deployment is via the distroless docker
image.

    docker build -t hassio-tar .

Unlike the shell script it doesn't decompress.  It purely handles decryption
only.  Rather than listing the contents with `tar -t` you'll need to include
gzip arguments like `tar -tz`.

```bash
tar -xOf backup.tar inner-addon.tar.gz | \
  docker run --rm -e HASSIO_PASSWORD -i hassio-tar | \
  tar -tz
```

For simplicity, I sometimes alias it as `htar`.

```bash
alias htar='docker run --rm -e HASSIO_PASSWORD -i hassio-tar'

# and later use the htar command
tar -xOf backup.tar inner-addon.tar.gz | \
  htar | \
  tar -tz
```
