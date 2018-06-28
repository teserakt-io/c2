package main

import (
	"github.com/dgraph-io/badger"
)

func (s *C2) deleteIdKey(id []byte) error {
	return dbDelete(s.dbi, id)
}

func (s *C2) deleteTopicKey(topic []byte) error {
	return dbDelete(s.dbt, topic)
}

func dbDelete(db *badger.DB, key []byte) error {

	_, err := dbGetValue(db, key)
	if err != nil {
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	return err
}

func (s *C2) insertIdKey(id, key []byte) error {
	return dbInsertErase(s.dbi, id, key)
}

func (s *C2) insertTopicKey(topic, key []byte) error {
	return dbInsertErase(s.dbt, topic, key)
}

func dbInsertErase(db *badger.DB, key, value []byte) error {
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return err
}

func (s *C2) getIdKey(id []byte) ([]byte, error) {
	return dbGetValue(s.dbi, id)
}

func (s *C2) getTopicKey(topichash []byte) ([]byte, error) {
	return dbGetValue(s.dbt, topichash)
}

func dbGetValue(db *badger.DB, key []byte) ([]byte, error) {
	var value []byte
	err := db.View(func(txn *badger.Txn) error {
		v, err := txn.Get(key)
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

func (s *C2) countIdKeys() (int, error) {
	return dbCountKeys(s.dbi)
}

func (s *C2) countTopicKeys() (int, error) {
	return dbCountKeys(s.dbt)
}

func dbCountKeys(db *badger.DB) (int, error) {

	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchSize = 10
	var count int
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(itOpts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
