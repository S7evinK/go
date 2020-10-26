// Copyright (c) 2020 Nikos Filippakis
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"math/rand"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// AESCTRKeyLength is the length of the AES256-CTR key used.
	AESCTRKeyLength = 32
	// AESCTRIVLength is the length of the AES256-CTR IV used.
	AESCTRIVLength = 16
	// HMACKeyLength is the length of the HMAC key used.
	HMACKeyLength = 32
	// SHAHashLength is the length of the SHA hash used.
	SHAHashLength = 32
)

// XorA256CTR encrypts the input with the keystream generated by the AES256-CTR algorithm with the given arguments.
func XorA256CTR(source []byte, key [AESCTRKeyLength]byte, iv [AESCTRIVLength]byte) []byte {
	block, _ := aes.NewCipher(key[:])
	result := make([]byte, len(source))
	cipher.NewCTR(block, iv[:]).XORKeyStream(result, source)
	return result
}

// GenAttachmentA256CTR generates a new random AES256-CTR key and IV suitable for encrypting attachments.
func GenAttachmentA256CTR() (key [AESCTRKeyLength]byte, iv [AESCTRIVLength]byte) {
	_, err := rand.Read(key[:])
	if err != nil {
		panic(err)
	}

	// The last 8 bytes of the IV act as the counter in AES-CTR, which means they're left empty here
	_, err = rand.Read(iv[:8])
	if err != nil {
		panic(err)
	}
	return
}

// GenA256CTRIV generates a random IV for AES256-CTR with the last bit set to zero.
func GenA256CTRIV() (iv [AESCTRIVLength]byte) {
	_, err := rand.Read(iv[:])
	if err != nil {
		panic(err)
	}
	iv[8] &= 0x7F
	return
}

// DeriveKeysSHA256 derives an AES and a HMAC key from the given recovery key.
func DeriveKeysSHA256(key []byte, name string) ([AESCTRKeyLength]byte, [HMACKeyLength]byte) {
	var zeroBytes [32]byte

	derivedHkdf := hkdf.New(sha256.New, key[:], zeroBytes[:], []byte(name))

	var aesKey [AESCTRKeyLength]byte
	var hmacKey [HMACKeyLength]byte
	derivedHkdf.Read(aesKey[:])
	derivedHkdf.Read(hmacKey[:])

	return aesKey, hmacKey
}

// PBKDF2SHA512 generates a key of the given bit-length using the given passphrase, salt and iteration count.
func PBKDF2SHA512(password []byte, salt []byte, iters int, keyLenBits int) []byte {
	return pbkdf2.Key(password, salt, iters, keyLenBits/8, sha512.New)
}

// DecodeBase58RecoveryKey recovers the secret storage from a recovery key.
func DecodeBase58RecoveryKey(recoveryKey string) []byte {
	noSpaces := strings.ReplaceAll(recoveryKey, " ", "")
	decoded := base58.Decode(noSpaces)
	if len(decoded) != AESCTRKeyLength+3 { // AESCTRKeyLength bytes key and 3 bytes prefix / parity
		return nil
	}
	var parity byte
	for _, b := range decoded[:34] {
		parity ^= b
	}
	if parity != decoded[34] || decoded[0] != 0x8B || decoded[1] != 1 {
		return nil
	}
	return decoded[2:34]
}

// EncodeBase58RecoveryKey recovers the secret storage from a recovery key.
func EncodeBase58RecoveryKey(key []byte) string {
	var inputBytes [35]byte
	copy(inputBytes[2:34], key[:])
	inputBytes[0] = 0x8B
	inputBytes[1] = 1

	var parity byte
	for _, b := range inputBytes[:34] {
		parity ^= b
	}
	inputBytes[34] = parity
	recoveryKey := base58.Encode(inputBytes[:])

	var spacedKey string
	for i, c := range recoveryKey {
		if i > 0 && i%4 == 0 {
			spacedKey += " "
		}
		spacedKey += string(c)
	}
	return spacedKey
}

// HMACSHA256B64 calculates the base64 of the SHA256 hmac of the input with the given key.
func HMACSHA256B64(input []byte, hmacKey [HMACKeyLength]byte) string {
	h := hmac.New(sha256.New, hmacKey[:])
	h.Write(input)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}