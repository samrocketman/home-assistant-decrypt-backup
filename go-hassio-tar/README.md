# hassio-tar - A Go-based version

- :white_check_mark: Small distroless docker image (<1MB)
- :white_check_mark: More secure than the shell script. This version uses
  in-memory decryption stream and non-root execution.
- :white_check_mark: Better integrity - There's a small decryption check
  verifying the decrypted size matches the expected size [reported by
  SecureTar][securetar] format.

# Pre-compiled releases

Pre-compiled [GitHub releases][releases].

[releases]: https://github.com/samrocketman/home-assistant-decrypt-backup/releases

# Compile yourself with Docker

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

[securetar]: https://github.com/pvizeli/securetar
