/*
Copyright 2025-2026 Sam Gleske - https://github.com/samrocketman/home-assistant-decrypt-backup/blob/main/LICENSE
Apache License - Version 2.0, January 2004
*/
// Package main - libsodium-compatible crypto_secretstream_xchacha20poly1305 decryption.
// Inlined from github.com/openziti/secretstream to avoid extra dependency.
// https://github.com/openziti/secretstream/blob/353218d728b15083b92de93e84b27d429a2917d0/LICENSE
// MIT License - Copyright (c) 2020 NetFoundry, Inc
// Uses only golang.org/x/crypto (chacha20, poly1305).

package main

import (
	"crypto/subtle"
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/poly1305"
)

const (
	secretstreamTagMessage = 0
	secretstreamTagFinal   = 0x03 // TagPush | TagRekey

	secretstreamHeaderBytes = 24
	secretstreamABytes      = 16 + 1
	secretstreamChunkSize   = 1024 * 1024 // 1 MiB
)

var (
	errSecretstreamInvalidKey   = errors.New("secretstream: invalid key")
	errSecretstreamInvalidInput = errors.New("secretstream: invalid input")
	errSecretstreamCryptoFailed = errors.New("secretstream: crypto failed")
)

type secretstreamDecryptor struct {
	k     [32]byte
	nonce [12]byte
	pad   [8]byte
}

func newSecretstreamDecryptor(key, header []byte) (*secretstreamDecryptor, error) {
	if len(key) != 32 || len(header) != secretstreamHeaderBytes {
		return nil, errSecretstreamInvalidKey
	}
	s := &secretstreamDecryptor{}
	k, err := chacha20.HChaCha20(key, header[:16])
	if err != nil {
		return nil, err
	}
	copy(s.k[:], k)
	s.reset()
	copy(s.nonce[4:], header[16:])
	return s, nil
}

func (s *secretstreamDecryptor) reset() {
	for i := range s.nonce {
		s.nonce[i] = 0
	}
	s.nonce[0] = 1
}

func (s *secretstreamDecryptor) pull(in []byte) ([]byte, byte, error) {
	if len(in) < secretstreamABytes {
		return nil, 0, errSecretstreamInvalidInput
	}
	mlen := len(in) - secretstreamABytes

	var block [64]byte
	var slen [8]byte
	var pad0 [16]byte

	chacha, err := chacha20.NewUnauthenticatedCipher(s.k[:], s.nonce[:])
	if err != nil {
		return nil, 0, err
	}
	chacha.XORKeyStream(block[:], block[:])

	var polyInit [32]byte
	copy(polyInit[:], block[:])
	poly := poly1305.New(&polyInit)

	for i := range block {
		block[i] = 0
	}
	block[0] = in[0]
	chacha.XORKeyStream(block[:], block[:])
	tag := block[0]
	block[0] = in[0]
	if _, err := poly.Write(block[:]); err != nil {
		return nil, 0, err
	}

	c := in[1:]
	if _, err := poly.Write(c[:mlen]); err != nil {
		return nil, 0, err
	}
	padlen := (0x10 - len(block) + mlen) & 0xf
	if _, err := poly.Write(pad0[:padlen]); err != nil {
		return nil, 0, err
	}
	binary.LittleEndian.PutUint64(slen[:], 0)
	if _, err := poly.Write(slen[:]); err != nil {
		return nil, 0, err
	}
	binary.LittleEndian.PutUint64(slen[:], uint64(len(block)+mlen))
	if _, err := poly.Write(slen[:]); err != nil {
		return nil, 0, err
	}

	mac := poly.Sum(nil)
	storedMac := c[mlen:]
	if subtle.ConstantTimeCompare(mac, storedMac) == 0 {
		return nil, 0, errSecretstreamCryptoFailed
	}

	m := make([]byte, mlen)
	chacha.XORKeyStream(m, c[:mlen])

	// XOR first 8 bytes of mac into nonce (libsodium INONCEBYTES)
	for i := 0; i < 8 && i < len(mac); i++ {
		s.nonce[4+i] ^= mac[i]
	}
	bufInc(s.nonce[:4])

	return m, tag, nil
}

func bufInc(n []byte) {
	c := 1
	for i := range n {
		c += int(n[i])
		n[i] = byte(c)
		c >>= 8
	}
}
