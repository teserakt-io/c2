package e4common

import (
	"crypto/rand"
	"encoding/binary"
	"time"
	"errors"
	"github.com/miscreant/miscreant/go"
	"golang.org/x/crypto/sha3"
)


func HashTopic(topic string) []byte {

	return hashStuff([]byte(topic))
}

func HashIdAlias(idalias string) []byte {

	return hashStuff([]byte(idalias))
}

func hashStuff(data []byte) []byte {
	h := sha3.Sum256(data)
	return h[:]
}

func Encrypt(key []byte, ad []byte, pt []byte) ([]byte, error) {

	c, err := miscreant.NewAESCMACSIV(key)
	if err != nil {
		return []byte{}, err
	}
	ads := make([][]byte, 1)
	ads[0] = ad
	return c.Seal(nil, pt, ads...)
}

func Decrypt(key []byte, ad []byte, ct []byte) ([]byte, error) {

	c, err := miscreant.NewAESCMACSIV(key)
	if err != nil {
		return []byte{}, err
	}
	if len(ct) < c.Overhead() {
		return []byte{}, errors.New("too short ciphertext")
	}
	ads := make([][]byte, 1)
	ads[0] = ad
	return c.Open(nil, ct, ads...)
}

func RandomKey() []byte {
	key := make([]byte, KeyLen)
	rand.Read(key)
	return key
}

func RandomId() []byte {
	id := make([]byte, IdLen)
	rand.Read(id)
	return id
}

func Protect(message[]byte, key []byte) ([]byte, error) {

	timestamp := make([]byte, TimestampLen)
	binary.LittleEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	ct, err := Encrypt(key, timestamp, message)
	if err != nil {
		return nil, err
	}
	protected := append(timestamp, ct...)

	return protected, nil
}

func Unprotect(protected []byte, key []byte) ([]byte, error) {

	ct := protected[TimestampLen:]
	timestamp := protected[:TimestampLen]

	ts := binary.LittleEndian.Uint64(timestamp)
	now := uint64(time.Now().Unix())
	if now < ts {
		return nil, errors.New("timestamp received is in the future")
	}
	if now-ts > MaxSecondsDelay {
		return nil, errors.New("timestamp too old")
	}

	pt, err := Decrypt(key, timestamp, ct)
	if err != nil {
		return nil, err
	}

	return pt, nil
}
