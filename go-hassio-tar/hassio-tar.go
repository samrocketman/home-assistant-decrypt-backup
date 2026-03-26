package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/gtank/blake2/blake2b"
	"golang.org/x/crypto/argon2"
)

// SecureTar format constants from securetar/__init__.py
const (
	secureTarMagicLen     = 9
	secureTarVersionByte  = 10
	secureTarReservedLen  = 6
	secureTarFileIDLen    = 16 // 9 + 1 + 6
	secureTarMetadataLen  = 16 // 8 plaintext size + 8 reserved
	secureTarV2SaltLen    = 16
	secureTarV2HeaderSize = 48 // 16 + 8 + 8 + 16

	// V3 constants
	v3RootSaltLen        = 16
	v3ValidationSaltLen  = 16
	v3ValidationKeyLen   = 32
	v3DerivationSaltLen  = 16
	v3SecretstreamHeader = 24
	v3CipherInitSize     = 16 + 16 + 32 + 16 + 24 // 104 bytes
	v3HeaderSize         = 16 + 16 + 104          // 136 bytes
	v3KDFOpsLimit        = 8
	v3KDFMemLimit        = 16 * 1024 // 16 MiB in KiB (argon2.IDKey takes KiB)
	v3Blake2bPerson      = "SecureTarv3"
)

var (
	secureTarMagicV2 = [16]byte{
		0x53, 0x65, 0x63, 0x75, 0x72, 0x65, 0x54, 0x61,
		0x72, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	secureTarMagicV3 = [16]byte{
		0x53, 0x65, 0x63, 0x75, 0x72, 0x65, 0x54, 0x61,
		0x72, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
)

func main() {
	// Read initial 16 bytes to detect format
	fileID := make([]byte, secureTarFileIDLen)
	n, err := io.ReadFull(os.Stdin, fileID)
	if err != nil && err != io.ErrUnexpectedEOF {
		if err == io.EOF {
			fmt.Fprintf(os.Stderr, "Error: Unexpected end of input\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error reading header: %v\n", err)
		}
		os.Exit(1)
	}
	if n < secureTarFileIDLen {
		_, _ = os.Stdout.Write(fileID[:n])
		os.Exit(0)
	}

	// Not SecureTar - pass through
	if !bytes.Equal(fileID[:secureTarMagicLen], []byte("SecureTar")) {
		_, _ = os.Stdout.Write(fileID)
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	version := fileID[9]
	if version != 2 && version != 3 {
		_, _ = os.Stdout.Write(fileID)
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	hassioPassword := os.Getenv("HASSIO_PASSWORD")
	if hassioPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: SecureTar found but HASSIO_PASSWORD not set.")
		os.Exit(1)
	}

	if version == 2 {
		decryptV2(fileID, hassioPassword)
		return
	}
	decryptV3(fileID, hassioPassword)
}

func decryptV2(fileID []byte, password string) {
	// Read rest of v2 header: 8 bytes expected size, 8 bytes zero, 16 bytes salt
	headerRest := make([]byte, secureTarV2HeaderSize-secureTarFileIDLen)
	n, err := io.ReadFull(os.Stdin, headerRest)
	if err != nil && err != io.ErrUnexpectedEOF {
		printError("v2 header", err)
	}
	if n < len(headerRest) {
		_, _ = os.Stdout.Write(fileID)
		_, _ = os.Stdout.Write(headerRest[:n])
		os.Exit(0)
	}

	if !bytes.Equal(fileID[:], secureTarMagicV2[:]) {
		_, _ = os.Stdout.Write(fileID)
		_, _ = os.Stdout.Write(headerRest)
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	expectedSize := binary.BigEndian.Uint64(headerRest[0:8])
	salt := headerRest[16:32]

	key, _ := sha256Iterating100Times([]byte(password))
	iv, _ := sha256Iterating100Times(append(key[:], salt...))
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cipher: %v\n", err)
		os.Exit(1)
	}
	stream := cipher.NewCBCDecrypter(block, iv)
	buffer := make([]byte, 1024)
	var decrypted []byte
	var decryptedSize uint64
	for {
		n, err = os.Stdin.Read(buffer)
		if err == io.EOF {
			break
		}
		if decrypted != nil {
			decryptedSize += uint64(len(decrypted))
			_, _ = os.Stdout.Write(decrypted)
		}
		if n > 0 {
			decrypted = make([]byte, n)
			stream.CryptBlocks(decrypted, buffer[:n])
		}
		if err != nil {
			printError("v2-1", err)
		}
	}
	if decrypted != nil && len(decrypted) > 0 {
		var unpadded []byte
		if expectedSize == decryptedSize+uint64(len(decrypted)) {
			unpadded = decrypted
		} else {
			unpadded = pkcs7Unpad(decrypted)
		}
		decryptedSize += uint64(len(unpadded))
		_, err = os.Stdout.Write(unpadded)
		if err != nil {
			printError("v2-2", err)
		}
	}
	if expectedSize != decryptedSize {
		printError("SecureTar", fmt.Errorf("expected %v bytes but decrypted %v bytes", expectedSize, decryptedSize))
	}
}

func decryptV3(fileID []byte, password string) {
	// Read metadata (plaintext size) and cipher init
	metadata := make([]byte, secureTarMetadataLen+v3CipherInitSize)
	n, err := io.ReadFull(os.Stdin, metadata)
	if err != nil && err != io.ErrUnexpectedEOF {
		printError("v3 header", err)
	}
	if n < len(metadata) {
		_, _ = os.Stdout.Write(fileID)
		_, _ = os.Stdout.Write(metadata[:n])
		os.Exit(0)
	}

	if !bytes.Equal(fileID[:], secureTarMagicV3[:]) {
		_, _ = os.Stdout.Write(fileID)
		_, _ = os.Stdout.Write(metadata)
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	plaintextSize := binary.BigEndian.Uint64(metadata[0:8])
	rootSalt := metadata[16 : 16+v3RootSaltLen]
	validationSalt := metadata[32 : 32+v3ValidationSaltLen]
	storedValidationKey := metadata[48 : 48+v3ValidationKeyLen]
	derivationSalt := metadata[80 : 80+v3DerivationSaltLen]
	secretstreamHeader := metadata[96 : 96+v3SecretstreamHeader]

	// Argon2id key derivation
	rootKey := argon2.IDKey(
		[]byte(password),
		rootSalt,
		v3KDFOpsLimit,
		v3KDFMemLimit,
		1, // threads
		32,
	)

	// Validation key for password verification
	validationKeyDigest, err := blake2b.NewDigest(rootKey, validationSalt, []byte(v3Blake2bPerson), 32)
	if err != nil {
		printError("v3 blake2b validation", err)
	}
	validationKey := validationKeyDigest.Sum(nil)

	if subtle.ConstantTimeCompare(validationKey, storedValidationKey) != 1 {
		fmt.Fprintln(os.Stderr, "ERROR: Invalid password for SecureTar v3.")
		os.Exit(1)
	}

	// Encryption key for secretstream
	encKeyDigest, err := blake2b.NewDigest(rootKey, derivationSalt, []byte(v3Blake2bPerson), 32)
	if err != nil {
		printError("v3 blake2b enc key", err)
	}
	encKey := encKeyDigest.Sum(nil)

	dec, err := newSecretstreamDecryptor(encKey, secretstreamHeader)
	if err != nil {
		printError("v3 secretstream init", err)
	}

	// Decrypt in chunks: 1 MiB plaintext + 17 bytes overhead per chunk
	chunkCipherSize := secretstreamChunkSize + secretstreamABytes
	buffer := make([]byte, chunkCipherSize)
	var written uint64
	for written < plaintextSize {
		toRead := chunkCipherSize
		remaining := plaintextSize - written
		if remaining <= secretstreamChunkSize {
			toRead = int(remaining) + secretstreamABytes
		}
		if toRead > len(buffer) {
			toRead = len(buffer)
		}
		n, err := io.ReadFull(os.Stdin, buffer[:toRead])
		if err != nil && err != io.ErrUnexpectedEOF {
			if err == io.EOF && n > 0 {
				// Partial read at EOF
			} else {
				printError("v3 read", err)
			}
		}
		if n == 0 {
			break
		}
		plain, tag, err := dec.pull(buffer[:n])
		if err != nil {
			printError("v3 decrypt", err)
		}
		_, err = os.Stdout.Write(plain)
		if err != nil {
			printError("v3 write", err)
		}
		written += uint64(len(plain))
		if tag == secretstreamTagFinal {
			break
		}
	}
	if written != plaintextSize {
		printError("SecureTar v3", fmt.Errorf("expected %v bytes but decrypted %v bytes", plaintextSize, written))
	}
}

func printError(traceID string, err error) {
	message := "================================================================================\n"
	message += "ERROR: %v %v\n"
	message += "================================================================================\n"
	fmt.Fprintf(os.Stderr, message, traceID, err)
	os.Exit(1)
}

func sha256Iterating100Times(input []byte) ([]byte, error) {
	hash := input
	for range 100 {
		hasher := sha256.New()
		_, err := hasher.Write(hash)
		if err != nil {
			return nil, err
		}
		hash = hasher.Sum(nil)
	}
	return hash[:16], nil
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return data
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return data
		}
	}
	return data[:len(data)-padding]
}
