package main

import (
	"encoding/hex"
	"errors"
	"github.com/jinzhu/gorm"
	e4 "teserakt/e4go/pkg/e4common"
)

// IDKey represents an Identity Key in the database given a unique device ID.
type IDKey struct {
	gorm.Model
	E4ID      [e4.IDLen]byte  `gorm:"unique;not null"`
	Key       [e4.KeyLen]byte `gorm:"not null"`
	TopicKeys []*TopicKey     `gorm:"many2many:idkeys_topickeys;"`
}

// TopicKey represents
type TopicKey struct {
	gorm.Model
	Topic  string          `gorm:"unique;not null"`
	Key    [e4.KeyLen]byte `gorm:"not null"`
	IDKeys []*IDKey        `gorm:"many2many:idkeys_topickeys;"`
}

// This function is responsible for the
func (s *C2) dbInitialize() error {
	s.logger.Log("msg", "Database Migration Started.")
	// TODO: better DB migration logic.
	s.db.AutoMigrate(&IDKey{})
	s.db.AutoMigrate(&TopicKey{})
	s.logger.Log("msg", "Database Migration Finished.")
	return nil
}

func (s *C2) insertIDKey(id, key []byte) error {
	var idbytes [e4.IDLen]byte
	var keybytes [e4.KeyLen]byte
	if len(id) != e4.IDLen {
		return errors.New("ID size incorrect, not 32 bytes")
	}
	if len(key) != e4.KeyLen {
		return errors.New("Key size not 64 bytes")
	}
	copy(idbytes[:], id)
	copy(keybytes[:], key)
	idkey := IDKey{E4ID: idbytes, Key: keybytes}
	if s.db.NewRecord(idkey) {
		s.db.Create(&idkey)
	} else {
		s.db.Model(&idkey).Updates(idkey)
	}
	// TODO: failures from GORM?
	return nil
}

func (s *C2) insertTopicKey(topic string, key []byte) error {
	var keybytes [e4.KeyLen]byte
	if len(key) != e4.KeyLen {
		return errors.New("Key size not 64 bytes")
	}
	copy(keybytes[:], key)
	topickey := TopicKey{Topic: topic, Key: keybytes}
	if s.db.NewRecord(topickey) {
		s.db.Create(&topickey)
	} else {
		s.db.Model(&topickey).Updates(topickey)
	}
	// TODO: failures from GORM?
	return nil
}

func (s *C2) getIDKey(id []byte) ([]byte, error) {
	var idkey IDKey
	s.db.Where("E4ID=?", id).First(&idkey)

	// TODO: return error when idkey.key == nil
	return idkey.Key[:], nil
}

func (s *C2) getTopicKey(topic string) ([]byte, error) {
	var topickey TopicKey
	s.db.Where("Topic=?", topic).First(&topickey)

	// TODO: return error when topickey.key == nil
	return topickey.Key[:], nil
}

func (s *C2) deleteIDKey(id []byte) error {
	var idbytes [e4.IDLen]byte
	if len(id) != e4.IDLen {
		return errors.New("ID size incorrect, not 32 bytes")
	}
	copy(idbytes[:], id)
	var idkey IDKey
	s.db.Where("E4ID=?", idbytes).First(&idkey)
	s.db.Model(&idkey).Association("TopicKeys").Clear()
	s.db.Delete(&idkey)
	return nil
}

func (s *C2) deleteTopicKey(topic string) error {
	var topicKey TopicKey
	s.db.Where("Topic=?", topic).First(&topicKey)
	s.db.Model(&topicKey).Association("IDKeys").Clear()
	s.db.Delete(&topicKey)
	return nil
}

func (s *C2) countIDKeys() (int, error) {
	var idkey IDKey
	var count int
	s.db.Model(&idkey).Count(&count)
	return count, nil
}

func (s *C2) countTopicKeys() (int, error) {
	var topickey TopicKey
	var count int
	s.db.Model(&topickey).Count(&count)
	return count, nil
}

func (s *C2) dbGetIDListHex() ([]string, error) {
	var idkeys []IDKey
	var hexids []string
	s.db.Find(&idkeys)

	for _, idkey := range idkeys {
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID[0:]))
	}

	return hexids, nil
}

func (s *C2) dbGetTopicsList() ([]string, error) {
	var topickeys []TopicKey
	var topics []string
	s.db.Find(&topickeys)

	for _, topickey := range topickeys {
		topics = append(topics, topickey.Topic)
	}

	return topics, nil
}
