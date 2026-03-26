# hassio-tar

Named `htar` or `hassio-tar`.  A simple utility to decrypt Home Assistant
backups similar to plain `tar`.

Pre-compiled binaries available from [releases page]; see under Assets.
Examples assume you download as `htar` within your shell `$PATH`.

# Example

## Set your decryption key

Set `HASSIO_PASSWORD` environment variable with your backup password.

```bash
export HASSIO_PASSWORD
read -ersp password:\  HASSIO_PASSWORD
```

> **Note**: If you are working from a dead drive backup, see my [HA password
> recovery guide].

## Decrypt your backup

Identify which inner tar file you wish to decrypt by using `tar` to list the
encrypted `tar.gz` files.

```bash
tar -tf backup.tar
```

List files within an encrypted `tar.gz`.

```bash
tar -xOf backup.tar inner-addon.tar.gz | \
  htar | \
  tar -tz
```

Extract files from an encrypted `tar.gz`.

```bash
tar -xOf backup.tar inner-addon.tar.gz | \
  htar | \
  tar -xz
```


# Build from source

Requires Go 1.23+.

```bash
go build -o htar ./go-hassio-tar
```

Or use GoReleaser to cross-compile all supported platforms:

```bash
make release-snapshot
```

# Testing

Tests run inside Docker for full isolation (works locally and in CI):

```bash
make test
```

# SecureTar support

Home Assistant uses a custom encryption format named [SecureTar].

| HA Backup format     | Supported?         | Home Assistant release |
| -------------------- | ------------------ | ---------------------- |
| SecureTar v1         | :x:                | not supported          |
| SecureTar v2         | :white_check_mark: | 2025.2.0 - 2026.2.x    |
| SecureTar v3         | :white_check_mark: | 2026.3.0 - latest      |

## Legacy shell script

If you are looking for the SecureTar v2 shell script, you may find it in the
[older 0.1.1 release].

# License

See [LICENSE](LICENSE).

    Copyright 2025-2026 Sam Gleske - https://github.com/samrocketman/home-assistant-decrypt-backup/blob/main/LICENSE
    Apache License - Version 2.0, January 2004

[HA password recovery guide]: https://github.com/samrocketman/blog/issues/90
[SecureTar]: https://github.com/home-assistant-libs/securetar
[older 0.1.1 release]: https://github.com/samrocketman/home-assistant-decrypt-backup/tree/v0.1.1
[releases page]: https://github.com/samrocketman/home-assistant-decrypt-backup/releases
