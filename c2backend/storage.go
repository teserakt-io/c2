package main

import (
	"github.com/dgraph-io/badger"
)

func (s *C2) dbDelete(key []byte) error {

	_, err := s.dbGetValue(key)
	if err != nil {
		return err
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	return err
}

func (s *C2) dbInsertErase(key, value []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return err
}

func (s *C2) dbInsertAppend(key, value []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	return err
}

func (s *C2) dbGetValue(key []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
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

func (s *C2) dbCountKeys() (int, error) {

	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchSize = 10
	var count int
	err := s.db.View(func(txn *badger.Txn) error {
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
