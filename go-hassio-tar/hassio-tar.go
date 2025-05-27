package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func main() {
	// SecureTarMagic = b"SecureTar\x02\x00\x00\x00\x00\x00\x00"
	var secureTarMagic [16]byte = [16]byte{
		0x53, 0x65, 0x63, 0x75, 0x72, 0x65, 0x54, 0x61,
		0x72, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	// SecureTar header is 48 bytes (16 magic, 16 don't care, 16 salt)
	const headerSize = 48

	// Create a buffer to hold the header
	header := make([]byte, headerSize)

	// Read SecureTar header from stdin
	n, err := io.ReadFull(os.Stdin, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		if err == io.EOF {
			fmt.Fprintf(os.Stderr, "Error: Unexpected end of input\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error reading header: %v\n", err)
		}
		os.Exit(1)
	}

	if n < headerSize {
		_, err = os.Stdout.Write(header)
		os.Exit(0)
	}

	if !bytes.Equal(header[:len(secureTarMagic)], secureTarMagic[:]) {
		_, err = os.Stdout.Write(header)
		_, err = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	hassioPassword := os.Getenv("HASSIO_PASSWORD")
	if hassioPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: SecureTar found but HASSIO_PASSWORD not set.")
		os.Exit(1)
	}
	key, _ := Sha256Iterating100Times([]byte(hassioPassword))
	// last 16 bytes of 48-byte header is salt
	iv, _ := Sha256Iterating100Times(append(key[:], header[32:]...))
	//fmt.Fprintf(os.Stderr, "key: %x\n", key)
	//fmt.Fprintf(os.Stderr, "iv: %x\n", iv)
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cipher: %v\n", err)
		os.Exit(1)
	}
	stream := cipher.NewCBCDecrypter(block, iv)
	buffer := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buffer)
		if n > 0 {
			// Decrypt the data
			decrypted := make([]byte, n)
			stream.CryptBlocks(decrypted, buffer[:n])
			unpadded := decrypted
			if n < 1024 {
				unpadded, err = pkcs7Unpad(decrypted)
				if err != nil {
					panic(err)
				}
			}
			// Write the decrypted data to stdout
			_, err = os.Stdout.Write(unpadded)
			if err != nil {
				panic(err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
	}
}

func Sha256Iterating100Times(input []byte) ([]byte, error) {
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

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("pkcs7: data is empty")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) {
		return nil, fmt.Errorf("pkcs7: invalid padding size greater than block size")
	}
	if padding == 0 {
		return nil, fmt.Errorf("pkcs7: invalid padding size zero")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return data, nil
			//return nil, fmt.Errorf("pkcs7: invalid padding byte")
		}
	}
	return data[:len(data)-padding], nil
}
