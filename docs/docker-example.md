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
