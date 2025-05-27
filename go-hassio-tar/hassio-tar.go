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
			stream.CryptBlocks(buffer[:n], buffer[:n])
			// Write the decrypted data to stdout
			if n < 1024 {
				output, err := removePadding(buffer[:n])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Cipher padding error: %v\n", err)
					os.Exit(1)
				}
				os.Stdout.Write(output)
			} else {
				os.Stdout.Write(buffer[:n])
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	}
}
func Sha256Iterating100Times(data []byte) ([]byte, error) {
	b := data

	for range 100 {
		h := sha256.New()
		if _, err := h.Write(b); err != nil {
			return nil, err
		}

		b = h.Sum(nil)
	}

	return b[:16], nil
}
func removePadding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	paddingLength := int(data[len(data)-1])
	if paddingLength > len(data) || paddingLength == 0 {
		return nil, fmt.Errorf("invalid padding length")
	}

	return data[:len(data)-paddingLength], nil
}
