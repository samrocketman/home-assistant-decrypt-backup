# EXPERIMENTAL hassio-tar

> This version has been known to fail.  For reliability you should only use the
> shell script `hassio-tar.sh`.

A simple tool which can decrypt Home Assistant SecureTar encrypted backups.

It will decrypt and decompress the backup.  It writes the tar output to stdout
which you can process with tar command yourself.

Works with encrypted tar directly or an inner tar within a Home Assistant
backup.

# Requirements

- Bash
- OpenSSL
- BSD or GNU core utils; or BusyBox

Set `HASSIO_PASSWORD` environment variable with your backup password.

    export HASSIO_PASSWORD
    read -ersp password: HASSIO_PASSWORD

# Examples

It can process encrypted tars directly.  For example, a backup of wireguard-ui
add-on.

    ./hassio-tar.sh ./c92fe070_wireguard-ui.tar.gz | tar -t
    mkdir wireguard-ui
    ./hassio-tar.sh ./c92fe070_wireguard-ui.tar.gz | tar -x -C ./wireguard-ui/

List the contents of a tar file (a Home Assistant Backup).

    ./hassio-tar.sh WireGuard_UI_1_2025-05-08_00.21_39444028.tar

Followed by decrypting and decrypting an inner tar within a Home Assistant
backup.

    ./hassio-tar.sh WireGuard_UI_1_2025-05-08_00.21_39444028.tar ./c92fe070_wireguard-ui.tar.gz

Because `hassio-tar.sh` handles streams, it can be used to manage only
decryption and decompression.  The following example uses `tar` to extract the
encrypted SecureTar, `hassio-tar.sh` to decrypt and decompress the SecureTar,
followed by tar to extract the add-on contents for inspection.

```bash
mkdir some-addon
tar -xOf your-backup.tar file.tar.gz  | \
  hassio-tar.sh | \
  tar -xC some-addon
```

# Docker Examples

`hassio-tar.sh` can process backups on stdin as well.  Start by building the
docker image example.

    docker build -t hassio-tar .

List the contents of a Home Assistant backup.

```bash
docker run \
  --rm \
  -i \
  -e HASSIO_PASSWORD hassio-tar \
  < WireGuard_UI_1_2025-05-08_00.21_39444028.tar
```

Decrypt an encrypted inner tar within a Home Assistant backup.
```bash
docker run \
  --rm \
  -i \
  -e HASSIO_PASSWORD \
  hassio-tar \
  c92fe070_wireguard-ui.tar.gz \
  < WireGuard_UI_1_2025-05-08_00.21_39444028.tar | tar -t
```


# Security Disclosure

OpenSSL CLI has a known limitation where the AES Key and IV are only supported
as command line arguments.

If another user on the system inspects process arguments (e.g. `ps aux`), then
the key and IV will be visible for the file being decrypted.

[The Go version is all in-memory and more secure](go-hassio-tar).
