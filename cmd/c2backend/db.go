package main

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/jinzhu/gorm"
)

// IDKey represents an Identity Key in the database given a unique device ID.
type IDKey struct {
	ID        int         `gorm:"primary_key:true"`
	E4ID      []byte      `gorm:"unique;NOT NULL"`
	Key       []byte      `gorm:"NOT NULL"`
	TopicKeys []*TopicKey `gorm:"many2many:idkeys_topickeys;"`
}

// TopicKey represents
type TopicKey struct {
	ID     int      `gorm:"primary_key:true"`
	Topic  string   `gorm:"unique;NOT NULL"`
	Key    []byte   `gorm:"NOT NULL"`
	IDKeys []*IDKey `gorm:"many2many:idkeys_topickeys;"`
}

// This function is responsible for the
func (s *C2) dbInitialize() error {
	s.logger.Log("msg", "Database Migration Started.")
	// TODO: better DB migration logic.
	// TODO: transactions?
	//tx := s.db.Begin()

	if result := s.db.AutoMigrate(&IDKey{}); result.Error != nil {
		//tx.Rollback()
		return result.Error
	}
	if result := s.db.AutoMigrate(&TopicKey{}); result.Error != nil {
		//tx.Rollback()
		return result.Error
	}
	/*if err := tx.Commit().Error; err != nil {
		return err
	}*/
	s.logger.Log("msg", "Database Migration Finished.")
	return nil
}

func (s *C2) insertIDKey(id, key []byte) error {
	idkey := IDKey{E4ID: id, Key: key}
	if s.db.NewRecord(idkey) {
		if result := s.db.Create(&idkey); result.Error != nil {
			return result.Error
		}
	} else {
		if result := s.db.Model(&idkey).Updates(idkey); result.Error != nil {
			return result.Error
		}
	}
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
	result := s.db.Where(&IDKey{}).First(&idkey)
	if gorm.IsRecordNotFoundError(result.Error) {
		// TODO: do we return an error for this?
		return nil, errors.New("ID not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if !bytes.Equal(id, idkey.E4ID) {
		return nil, errors.New("Internal error: struct not populated but GORM indicated success")
	}
	return idkey.Key[:], nil
}

func (s *C2) getTopicKey(topic string) ([]byte, error) {
	var topickey TopicKey
	result := s.db.Where(&TopicKey{Topic: topic}).First(&topickey)
	if gorm.IsRecordNotFoundError(result.Error) {
		return nil, errors.New("Topic not found")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if topickey.Topic != topic {
		return nil, errors.New("Internal error: struct not populated but GORM indicated success")
	}
	return topickey.Key[:], nil
}

func (s *C2) deleteIDKey(id []byte) error {
	var idkey IDKey

	if result := s.db.Where(&IDKey{E4ID: id}).First(&idkey); result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			return errors.New("ID not found, nothing to delete")
		} else {
			return result.Error
		}
	}
	// safety check:
	if !bytes.Equal(idkey.E4ID, id) {
		return errors.New("Single record not populated correctly; preventing whole DB delete")
	}
	tx := s.db.Begin()
	if result := tx.Model(&idkey).Association("TopicKeys").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := tx.Delete(&idkey); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *C2) deleteTopicKey(topic string) error {
	var topicKey TopicKey
	if result := s.db.Where(&TopicKey{Topic: topic}).First(&topicKey); result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			return errors.New("ID not found, nothing to delete")
		} else {
			return result.Error
		}
	}
	if topicKey.Topic != topic {
		return errors.New("Single record not populated correctly; preventing whole DB delete")
	}
	tx := s.db.Begin()
	if result := s.db.Model(&topicKey).Association("IDKeys").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := s.db.Delete(&topicKey); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *C2) countIDKeys() (int, error) {
	var idkey IDKey
	var count int
	if result := s.db.Model(&idkey).Count(&count); result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (s *C2) countTopicKeys() (int, error) {
	var topickey TopicKey
	var count int
	if result := s.db.Model(&topickey).Count(&count); result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (s *C2) dbGetIDListHex() ([]string, error) {
	var idkeys []IDKey
	var hexids []string
	if result := s.db.Find(&idkeys); result.Error != nil {
		return nil, result.Error
	}

	for _, idkey := range idkeys {
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID[0:]))
	}

	return hexids, nil
}

func (s *C2) dbGetTopicsList() ([]string, error) {
	var topickeys []TopicKey
	var topics []string
	if result := s.db.Find(&topickeys); result.Error != nil {
		return nil, result.Error
	}

	for _, topickey := range topickeys {
		topics = append(topics, topickey.Topic)
	}

	return topics, nil
}

/* -- M2M Functions -- */

// This function links a topic and an id/key. The link is created in both
// directions (IDkey to Topics, Topic to IDkeys).
func (s *C2) linkIDTopic(id []byte, topic string) error {

	var idkey IDKey
	var topickey TopicKey

	if err := s.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot link to topic")
		} else {
			return err
		}
	}
	if err := s.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot link to IDkey")
		} else {
			return err
		}
	}

	tx := s.db.Begin()

	if err := tx.Model(&idkey).Association("TopicKeys").Append(&topickey).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

// This function removes the relationship between a Topic and an ID, but
// does not delete the Topic or the ID.
func (s *C2) unlinkIDTopic(id []byte, topic string) error {

	var idkey IDKey
	var topickey TopicKey

	if err := s.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot unlink from topic")
		} else {
			return err
		}
	}
	if err := s.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot unlink from IDkey")
		} else {
			return err
		}
	}

	tx := s.db.Begin()

	if err := tx.Model(&idkey).Association("TopicKeys").Delete(&topickey).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *C2) countTopicsForID(id []byte) (int, error) {

	var idkey IDKey

	if err := s.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, errors.New("ID Key not found, cannot link to topic")
		} else {
			return 0, err
		}
	}

	count := s.db.Model(&idkey).Association("TopicKeys").Count()
	return count, nil
}

func (s *C2) getTopicsForID(id []byte, offset int, count int) ([]string, error) {

	var idkey IDKey
	var topickeys []TopicKey
	var topics []string

	if err := s.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errors.New("ID Key not found, cannot link to topic")
		} else {
			return nil, err
		}
	}

	if err := s.db.Model(&idkey).Offset(offset).Limit(count).Association("TopicKeys").Find(&topickeys).Error; err != nil {
		return nil, err
	}

	for _, topickey := range topickeys {
		topics = append(topics, topickey.Topic)
	}

	return topics, nil
}

func (s *C2) countIDsForTopic(topic string) (int, error) {
	var topickey TopicKey

	if err := s.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, errors.New("Topic key not found, cannot link to ID")
		} else {
			return 0, err
		}
	}

	count := s.db.Model(&topickey).Association("IDKeys").Count()
	return count, nil
}

func (s *C2) getIdsforTopic(topic string, offset int, count int) ([]string, error) {

	var topickey TopicKey
	var idkeys []IDKey
	var hexids []string

	if err := s.db.Where(&TopicKey{Topic: topic}).First(&topic).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errors.New("ID Key not found, cannot link to topic")
		} else {
			return nil, err
		}
	}

	if err := s.db.Model(&topickey).Offset(offset).Limit(count).Association("IDKeys").Find(&idkeys).Error; err != nil {
		return nil, err
	}

	for _, idkey := range idkeys {
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID[:]))
	}

	return hexids, nil
}
