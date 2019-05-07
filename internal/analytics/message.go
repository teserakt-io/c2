package analytics

import (
	"bytes"
)

// LoggedMessage defines a type holding the data to be logged on C2 messages
type LoggedMessage struct {
	Duplicate       bool   `json:"duplicate"`
	Qos             byte   `json:"qos"`
	Retained        bool   `json:"retained"`
	Topic           string `json:"topic"`
	MessageID       uint16 `json:"messageid"`
	Payload         []byte `json:"payload"`
	LooksEncrypted  bool   `json:"looksencrypted"`
	LooksCompressed bool   `json:"lookscompressed"`
	IsBase64        bool   `json:"isbase64"`
	IsUTF8          bool   `json:"isutf8"`
	IsJSON          bool   `json:"isjson"`
}

// LooksEncrypted indicate whenever given data looks encrypted or not.
func LooksEncrypted(data []byte) bool {
	// efficient, lazy heuristic, FN/FP-prone
	// will fail if e.g. ciphertext is prepended with non-random nonce
	if len(data) < 16 {
		// make the assumption that <16-byte data won't be encrypted
		return false
	}
	counter := make(map[int]int)
	for i := range data[:16] {
		counter[int(data[i])]++
	}
	if len(counter) < 10 {
		return false
	}
	// if encrypted, fails with low prob

	return true
}

// LooksCompressed indicate whenever given data looks compressed or not
func LooksCompressed(data []byte) bool {
	// application/zip
	if bytes.Equal(data[:4], []byte("\x50\x4b\x03\x04")) {
		return true
	}

	// application/x-gzip
	if bytes.Equal(data[:3], []byte("\x1F\x8B\x08")) {
		return true
	}

	// application/x-rar-compressed
	if bytes.Equal(data[:7], []byte("\x52\x61\x72\x20\x1A\x07\x00")) {
		return true
	}

	// zlib no/low compression
	if bytes.Equal(data[:2], []byte("\x78\x01")) {
		return true
	}

	// zlib default compression
	if bytes.Equal(data[:2], []byte("\x78\x9c")) {
		return true
	}

	// zlib best compression
	if bytes.Equal(data[:2], []byte("\x78\xda")) {
		return true
	}

	return false
}
