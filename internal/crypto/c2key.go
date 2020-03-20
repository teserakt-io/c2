// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crypto

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=c2key_mocks.go -package crypto -self_package github.com/teserakt-io/c2/internal/crypto github.com/teserakt-io/c2/internal/crypto C2KeyRotationTx

import (
	"errors"
	"fmt"
	"os"

	e4crypto "github.com/teserakt-io/e4go/crypto"
	"golang.org/x/crypto/curve25519"
)

// C2KeyRotationTx defines a C2 key rotation transaction
// It allows to access the newly generated C2 public key before applying it to the E4Key.
// A backup of the current key is made on transaction creation, and removed if it get rollbacked.
// When committed, the e4Key will be updated with the new C2 key pair, and the current key file will
// be overwritten with the new private key bytes.
type C2KeyRotationTx interface {
	// GetNewPublicKey returns the future Curve25519 public key of the C2
	// which is not yet applied to the E4Key.
	GetNewPublicKey() e4crypto.Curve25519PublicKey
	// Commit will replaces the current E4Key public and private C2 keys by the new ones.
	Commit() error
	// Rollback allows to restore the current key into the key file with the
	Rollback() error
}

type c2KeyRotationTx struct {
	e4Key           *e4PubKey
	newC2PrivateKey e4crypto.Curve25519PrivateKey
	newC2PublicKey  e4crypto.Curve25519PublicKey
	backupPath      string
	rolledBack      bool
}

var _ C2KeyRotationTx = (*c2KeyRotationTx)(nil)

func newC2KeyRotationTx(k *e4PubKey) (C2KeyRotationTx, error) {
	backupPath, err := k.backupCurrentC2Key()
	if err != nil {
		return nil, fmt.Errorf("failed to backup current C2 key: %v", err)
	}

	newC2Key := e4crypto.RandomKey()
	newC2PubKey, err := curve25519.X25519(newC2Key, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	return &c2KeyRotationTx{
		e4Key:           k,
		newC2PublicKey:  newC2PubKey,
		newC2PrivateKey: newC2Key,
		backupPath:      backupPath,
	}, nil
}

func (tx *c2KeyRotationTx) Commit() error {
	if tx.rolledBack {
		return errors.New("transaction have been rolled back")
	}
	if err := tx.e4Key.overwriteC2PrivateKey(tx.newC2PrivateKey); err != nil {
		return err
	}

	tx.e4Key.c2PrivKey = tx.newC2PrivateKey
	tx.e4Key.c2PubKey = tx.newC2PublicKey

	return nil
}

func (tx *c2KeyRotationTx) Rollback() error {
	if err := tx.e4Key.overwriteC2PrivateKey(tx.e4Key.c2PrivKey); err != nil {
		return err
	}

	if err := os.Remove(tx.backupPath); err != nil {
		return err
	}

	tx.rolledBack = true

	return nil
}

func (tx *c2KeyRotationTx) GetNewPublicKey() e4crypto.Curve25519PublicKey {
	return tx.newC2PublicKey
}
