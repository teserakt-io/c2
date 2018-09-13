package main

import (
	"encoding/hex"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
)

// IDKey represents an Identity Key in the database given a unique device ID.
type IDKey struct {
	gorm.Model
	ID        [16]byte `gorm:"primary_key:true,unique;not null"`
	Key       [32]byte `gorm:"not null"`
	TopicKeys []*TopicKey gorm:`gorm:"many2many:idkeys_topickeys;"`
}

// TopicKey represents
type TopicKey struct {
	gorm.Model
	TopicID [16]byte `gorm:"primary_key:true,unique;not null"`
	Key     [32]byte `gorm:"not null"`
	IDKeys  []*TopicKey gorm:`gorm:"many2many:idkeys_topickeys;"`
}

func (s *C2) createIDKey(id []byte) error {

}

func (s *C2) deleteIDKey(id []byte) error {

}

// TODO: Everything below here is deprecated and should be removed once the
// logic is moved over to GORM.
// signalling the type of key in the k-v store
const (
	IDByte    = 0
	TopicByte = 1
)

func (s *C2) deleteIDKey(id []byte) error {
	dbkey := append([]byte{IDByte}, id...)
	return s.dbDelete(dbkey)
}

func (s *C2) deleteTopicKey(topic string) error {
	dbkey := append([]byte{TopicByte}, []byte(topic)...)
	return s.dbDelete(dbkey)
}

func (s *C2) dbDelete(dbkey []byte) error {

	_, err := s.dbGetValue(dbkey)
	if err != nil {
		return err
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(dbkey)
	})
	return err
}

func (s *C2) insertIDKey(id, key []byte) error {
	dbkey := append([]byte{IDByte}, id...)
	return s.dbInsertErase(dbkey, key)
}

func (s *C2) insertTopicKey(topic string, key []byte) error {
	dbkey := append([]byte{TopicByte}, []byte(topic)...)
	return s.dbInsertErase(dbkey, key)
}

func (s *C2) dbInsertErase(dbkey, value []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(dbkey, value)
	})
	return err
}

func (s *C2) getIDKey(id []byte) ([]byte, error) {
	dbkey := append([]byte{IDByte}, id...)
	return s.dbGetValue(dbkey)
}

func (s *C2) getTopicKey(topic string) ([]byte, error) {
	dbkey := append([]byte{TopicByte}, []byte(topic)...)
	return s.dbGetValue(dbkey)
}

func (s *C2) dbGetValue(dbkey []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		v, err := txn.Get(dbkey)
		if err != nil {
			return err
		}
		value, err = v.Value()
		return err
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *C2) dbGetIDListHex() ([]string, error) {
	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchSize = 10
	var ids []string
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(itOpts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			dbkey := item.Key()
			if dbkey[0] == IDByte {
				ids = append(ids, hex.EncodeToString(dbkey[1:]))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *C2) dbGetTopicsList() ([]string, error) {
	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchSize = 10
	var topics []string
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(itOpts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			dbkey := item.Key()
			if dbkey[0] == TopicByte {
				topics = append(topics, string(dbkey[1:]))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func (s *C2) countIDKeys() (int, error) {
	return s.dbCountKeys(IDByte)
}

func (s *C2) countTopicKeys() (int, error) {
	return s.dbCountKeys(TopicByte)
}

func (s *C2) dbCountKeys(b byte) (int, error) {

	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchSize = 10
	var count int
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(itOpts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			dbkey := item.Key()
			if dbkey[0] == b {
				count++
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
