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

package models

import (
	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

// Client represents an Identity Key in the database given a unique device ID.
type Client struct {
	ID        int         `gorm:"primary_key:true"`
	E4ID      []byte      `gorm:"unique_index;NOT NULL"`
	Name      string      `gorm:"unique_index;NOT NULL" sql:"size:256"`
	Key       []byte      `gorm:"NOT NULL"`
	TopicKeys []*TopicKey `gorm:"many2many:clients_topickeys;"`
	Clients   []*Client   `gorm:"many2many:clients_clientkeys;association_jointable_foreignkey:clientkey_id"`
}

// TopicKey represents
type TopicKey struct {
	ID      int       `gorm:"primary_key:true"`
	Topic   string    `gorm:"unique_index;NOT NULL"`
	Key     []byte    `gorm:"NOT NULL"`
	Clients []*Client `gorm:"many2many:clients_topickeys;"`
}

// Hash return the E4 Hashed topic of the current TopicKey
func (t TopicKey) Hash() []byte {
	return e4crypto.HashTopic(t.Topic)
}

// DecryptKey returns the decrypted key of the current TopicKey
func (t TopicKey) DecryptKey(dbEncKey []byte) ([]byte, error) {
	key, err := e4crypto.Decrypt(dbEncKey, nil, t.Key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// DecryptKey returns the decrypted key of current Client
func (i Client) DecryptKey(dbEncKey []byte) ([]byte, error) {
	key, err := e4crypto.Decrypt(dbEncKey, nil, i.Key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// Topic returns the E4 Topic for the current Client
func (i Client) Topic() string {
	return e4.TopicForID(i.E4ID)
}
