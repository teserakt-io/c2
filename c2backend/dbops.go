package main

import (
	"github.com/dgraph-io/badger"
	"fmt"
)

const (
	IDByte = 0
	TopicByte = 1
)


func (s *C2) deleteIDKey(id []byte) error {
	dbkey := append(id, IDByte)
	return s.dbDelete(dbkey)
}

func (s *C2) deleteTopicKey(topic string) error {
	dbkey := append([]byte(topic), TopicByte)
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
	dbkey := append(id, IDByte)
	return s.dbInsertErase(dbkey, key)
}

func (s *C2) insertTopicKey(topic string, key []byte) error {
	dbkey := append([]byte(topic), TopicByte)
	return s.dbInsertErase(dbkey, key)
}

func (s *C2) dbInsertErase(dbkey, value []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(dbkey, value)
	})
	return err
}

func (s *C2) getIDKey(id []byte) ([]byte, error) {
	dbkey := append(id, IDByte)
	return s.dbGetValue(dbkey)
}

func (s *C2) getTopicKey(topic string) ([]byte, error) {
	dbkey := append([]byte(topic), TopicByte)
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
			dbkey, err := item.Value()
			if err != nil {
				return err
			}
			//if dbkey[32] == b {
				count++
				fmt.Printf("go %d\n", dbkey[32])
			//}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
