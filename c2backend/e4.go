package main

import (
	"encoding/hex"
	"errors"
	"log"
)

func (s *C2) newClient(id, key []byte) error {

	err := s.insertIDKey(id, key)
	if err != nil {
		log.Print(err)
		return errors.New("db update failed")
	}
	log.Printf("added client %s", hex.EncodeToString(id))
	return nil
}

// local
func (s *C2) removeClient(id []byte) error {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.deleteIDKey(in.Id)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "deletion error"}, nil
	}

	log.Printf("removed client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}