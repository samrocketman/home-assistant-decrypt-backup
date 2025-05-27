#!/bin/bash
# Created by Sam Gleske
# Sat May 24 09:50:01 AM EDT 2025
# Pop!_OS 22.04 LTS
# Linux 6.9.3-76060903-generic x86_64
# GNU bash, version 5.1.16(1)-release (x86_64-pc-linux-gnu)
#
# DESCRIPTION
#   A plain shell script which sticks to coreutils and openssl utilities in
#   order to decrypt and perform operations on a home assistant encrypted
#   secure tar.
# REQUIREMENTS
#   - bash
#   - openssl
#   - coreutils or busybox
# SECURITY DISCLOSURE
#   OpenSSL CLI has a known limitation where the AES Key and IV are only
#   supported as command line arguments.  This exposes the key and IV to
#   decrypt a single file inside of your Home Assistant Backup if another user
#   runs `ps aux` at a time when decryption is occurring.

helptext() {
cat <<EOF
SYNOPSIS
  ${0##*/} help|-h
  ${0##*/} BACKUP_TAR [INNER_ENCRYPTED_TAR]
  ${0##*/} ENCRYPTED_TAR | tar ...
  ${0##*/} < BACKUP_TAR
  ${0##*/} ENCRYPTED_TAR < BACKUP_TAR | tar ...
  ${0##*/} < ENCRYPTED_TAR | tar ...

DESCRIPTION
  A minimal utility for interacting with encrypted Home Assistant Backups.
  This utility supports both encrypted and plain backups.

  Use this CLI command to read encrypted tar and it will output tar format.
  Use standard tar command to process the output.

ARGUMENTS
  BACKUP_TAR
    A home assistant backup with encrypted or plain tars.  It contains a
    backup.json with information about the backup.
  ENCRYPTED_TAR or INNER_ENCRYPTED_TAR
    A SecureTar encrypted file.  It is an AES-128-CBC encrypted tar.gz file.

ENVIRONMENT VARIABLES
  HASSIO_PASSWORD
    Home Assistant encrypted backup password.

EXAMPLES
  List the contents of a backup tar.
    ${0##*/} some-backup.tar

  Extract decrypt and decompress a backup inner tar (use tar to processes).
    export HASSIO_PASSWORD
    read -ersp password: HASSIO_PASSWORD
    ${0##*/} some-backup.tar some-add-on.tar.gz | tar -t
EOF
}
bin_to_hex() {
  xxd -p | tr -d '\n'
}
hex_to_bin() {
  xxd -r -p
}
is_securetar() {
  # https://github.com/pvizeli/securetar/blob/main/securetar/__init__.py
  # SECURETAR_MAGIC = b"SecureTar\x02\x00\x00\x00\x00\x00\x00"
  local SECURETAR_MAGIC="53656375726554617202000000000000"
  [ "$(dd if="$1" bs=16 count=1 status=none | bin_to_hex)" = "$SECURETAR_MAGIC" ]
}
is_gzip() {
  local GZIP_MAGIC=1f8b08
  [ "$(dd if="$1" bs=3 count=1 status=none | bin_to_hex)" = "$GZIP_MAGIC" ]
}
is_tar() {
  # PaxTar magic (first 14 bytes) b"././@PaxHeader"
  local PAXTAR_MAGIC=2e2f2e2f40506178486561646572
  # ustar hex (skip 257 bytes and read 5 bytes)
  local USTAR_MAGIC=7573746172
  [ "$(dd if="$1" skip=257 bs=1 count=5 status=none | bin_to_hex)" = "$USTAR_MAGIC" ] ||
  [ "$(dd if="$1" bs=14 count=1 status=none | bin_to_hex)" = "$PAXTAR_MAGIC" ]
}
hash() {
  sha256sum | head -c64
}
derive_key() {
  # 100 iterations of sha256
  local data
  data="$(echo -n "$1" | hash)"
  for ignored in {0..98}; do
    data="$(echo -n "$data" | hex_to_bin | hash)"
  done
  echo "$data" | head -c32
}
derive_iv() {
  # 100 iterations of sha256
  local data
  data="$1"
  for ignored in {0..99}; do
    data="$(echo -n "$data" | hex_to_bin | hash)"
  done
  echo "$data" | head -c32
}
decrypt_stream() {
  local saltbytes salt key iv
  saltbytes=16
  salt="$(dd skip=2 bs="$saltbytes" count=1 status=none | xxd -p | tr -d '\n')"
  key="$(derive_key "${HASSIO_PASSWORD}")"
  iv="$(derive_iv "${key}${salt}")"
  dd bs="1M" status=none | \
    openssl enc -d -aes-128-cbc -K "$key" -iv "$iv"
}

#
# MAIN
#
for x in "$@"; do
  case "$x" in
    help|--help|-h)
      helptext
      exit
      ;;
  esac
done
if [ "$#" -gt 2 ]; then
  echo 'WARNING: expecting only two or less arguments but found '"$#." >&2
fi
if [ "$#" = 2 ]; then
  outer_tar="$1"
  inner_tar="$2"
  shift #shift only once intentional
elif [ "$#" = 1 ] && read -t 0; then
  inner_tar="$1"
elif [ "$#" = 1 ]; then
  outer_tar="$1"
  shift
fi
if [ -n "${outer_tar:-}" ]; then
  "$0" "$@" < "${outer_tar}"
  exit $?
fi

# Create a secure tmp scratch space for binary inspection
old_umask="$(umask)"
umask 0077
trap '[ ! -d "${TMP_DIR:-}" ] || rm -rf "${TMP_DIR}"' EXIT
TMP_DIR="$([ -w /dev/shm ] && mktemp -d -p /dev/shm || mktemp -d)"
export TMP_DIR
umask "$old_umask"
unset old_umask
# End secure tmp scratch space

header_file="${TMP_DIR}/outer_header"
dd of="$header_file" bs=272 count=1 status=none
if is_tar "$header_file"; then
  if [ -n "${inner_tar:-}" ]; then
    export skip_tar_list
    skip_tar_list=1
    { cat "$header_file"; cat; } | tar -xO "${inner_tar}" | "$0"
  elif [ -z "${skip_tar_list:-}" ]; then
    { cat "$header_file"; cat; } | tar -t
  else
    cat
  fi
  #echo 'ERROR: file appears to be a plain tar.' >&2
  #exit 1
elif is_securetar "$header_file"; then
  if [ -z "${HASSIO_PASSWORD:-}" ]; then
    echo 'ERROR: SecureTar found but HASSIO_PASSWORD not set.' >&2
    exit 1
  fi
  { cat "$header_file"; cat; } | decrypt_stream | "$0"
elif is_gzip "$header_file"; then
  { cat "$header_file"; cat; } | gzip -d -c
else
  echo 'ERROR: Unknown format.' >&2
  exit 1
fi
