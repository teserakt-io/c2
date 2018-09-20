package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	//	"github.com/jinzhu/gorm"
)

// IDKey represents an Identity Key in the database given a unique device ID.
type IDKey struct {
	ID        int         `gorm:"primary_key:true"`
	E4ID      []byte      `gorm:"unique;not null"`
	Key       []byte      `gorm:"not null"`
	TopicKeys []*TopicKey `gorm:"many2many:idkeys_topickeys;"`
}

// TopicKey represents
type TopicKey struct {
	ID     int      `gorm:"primary_key:true"`
	Topic  string   `gorm:"unique;not null"`
	Key    []byte   `gorm:"not null"`
	IDKeys []*IDKey `gorm:"many2many:idkeys_topickeys;"`
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
	idkey := IDKey{E4ID: id, Key: key}
	if s.db.NewRecord(idkey) {
		s.db.Create(&idkey)
	} else {
		s.db.Model(&idkey).Updates(idkey)
	}
	// TODO: failures from GORM?
	return nil
}

func (s *C2) insertTopicKey(topic string, key []byte) error {
	topickey := TopicKey{Topic: topic, Key: key}
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
	var idkey IDKey
	s.db.Where(&IDKey{E4ID: id}).First(&idkey)
	if !bytes.Equal(idkey.E4ID, id) {
		return errors.New("Unable to find single record; preventing whole DB delete")
	}
	s.db.Model(&idkey).Association("TopicKeys").Clear()
	s.db.Delete(&idkey)
	return nil
}

func (s *C2) deleteTopicKey(topic string) error {
	var topicKey TopicKey
	s.db.Where(&TopicKey{Topic: topic}).First(&topicKey)
	if topicKey.Topic != topic {
		return errors.New("Unable to find single record; preventing whole DB delete")
	}
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
