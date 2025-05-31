package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
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
	// SecureTar header: magic (16 bytes), expected size (8 bytes), zero (8 bytes), salt (16 bytes)
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
	// SecureTar from this point onward
	hassioPassword := os.Getenv("HASSIO_PASSWORD")
	if hassioPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: SecureTar found but HASSIO_PASSWORD not set.")
		os.Exit(1)
	}
	var expected_data_size uint64
	expected_data_size = binary.BigEndian.Uint64(header[16:24])
	key, _ := sha256Iterating100Times([]byte(hassioPassword))
	iv, _ := sha256Iterating100Times(append(key[:], header[32:]...))
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cipher: %v\n", err)
		os.Exit(1)
	}
	stream := cipher.NewCBCDecrypter(block, iv)
	buffer := make([]byte, 1024)
	var decrypted []byte
	decrypted = nil
	var decrypted_size uint64
	decrypted_size = 0
	for {
		n, err = os.Stdin.Read(buffer)
		if err == io.EOF {
			break
		}
		if decrypted != nil {
			decrypted_size += uint64(len(decrypted))
			_, err = os.Stdout.Write(decrypted)
		}
		if n > 0 {
			// Decrypt the data
			decrypted = make([]byte, n)
			stream.CryptBlocks(decrypted, buffer[:n])
		}
		if err != nil {
			printError("1 -", err)
		}
	}
	if decrypted != nil && len(decrypted) > 0 {
		// Process the remaining decrypted data
		var unpadded []byte
		if expected_data_size == (decrypted_size + uint64(len(decrypted))) {
			unpadded = decrypted
		} else {
			unpadded = pkcs7Unpad(decrypted[:])
		}
		decrypted_size += uint64(len(unpadded))
		_, err = os.Stdout.Write(unpadded)
		if err != nil {
			printError("2 -", err)
		}
	}
	if expected_data_size != decrypted_size {
		printError("SecureTar", fmt.Errorf("expected %v bytes but decrypted %v bytes", expected_data_size, decrypted_size))
	}
}
func printError(traceid string, err error) {
	message := "================================================================================\n"
	message += "ERROR: %v %v\n"
	message += message[:81]
	fmt.Fprintf(os.Stderr, message, traceid, err)
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
	padding := int(data[len(data)-1])
	// padding > len(data) -> return nil, fmt.Errorf("pkcs7: invalid padding size greater than block size")
	// padding == 0 -> return nil, fmt.Errorf("pkcs7: invalid padding size zero")
	if padding > len(data) || padding == 0 {
		return data
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return data
			//return nil, fmt.Errorf("pkcs7: invalid padding byte")
		}
	}
	return data[:len(data)-padding]
}
