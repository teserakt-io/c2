package main

import (
	"encoding/hex"
	"log"

	e4 "teserakt/e4common"
)

func (s *C2) newClient(id, key []byte) error {

	err := s.insertIDKey(id, key)
	if err != nil {
		log.Print("insertIDKey failed in newClient: ", err)
		return err
	}
	log.Printf("added client %s", hex.EncodeToString(id))
	return nil
}

func (s *C2) removeClient(id []byte) error {

	err := s.deleteIDKey(id)
	if err != nil {
		log.Print("deleteIDKey failed in removeClient: ", err)
		return err
	}

	log.Printf("removed client %s", hex.EncodeToString(id))
	return nil
}

func (s *C2) newTopicClient(id []byte, topic string) error {

	key, err := s.getTopicKey(topic)
	if err != nil {
		log.Print("getTopicKey failed in newTopicClient: ", err)
		return err
	}

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.SetTopicKey, topichash, key, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in newTopicClient: ", err)
		return err
	}
	err = s.sendToClient(id, payload)
	if err != nil {
		log.Print(err)
		return err
	}

	log.Printf("added topic '%s' to client %s", topic, hex.EncodeToString(id))
	return nil
}

func (s *C2) removeTopicClient(id []byte, topic string) error {

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.RemoveTopic, topichash, nil, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in removeTopicClient: ", err)
		return err
	}
	err = s.sendToClient(id, payload)
	if err != nil {
		log.Print("sendToClient failed in removeTopicClient", err)
		return err
	}

	log.Printf("removed topic '%s' from client %s", topic, hex.EncodeToString(id))
	return nil
}

func (s *C2) resetClient(id []byte) error {

	payload, err := s.CreateAndProtectForID(e4.ResetTopics, nil, nil, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in resetClient: ", err)
		return err
	}
	err = s.sendToClient(id, payload)
	if err != nil {
		log.Print("sendToClient failed in resetClient: ", err)
		return err
	}

	log.Printf("reset client %s", hex.EncodeToString(id))
	return nil
}

func (s *C2) newTopic(topic string) error {

	key := e4.RandomKey()

	err := s.insertTopicKey(topic, key)
	if err != nil {
		log.Print("insertTopicKey failed in newTopic: ", err)
		return err
	}
	log.Printf("added topic %s", topic)
	return nil
}

func (s *C2) removeTopic(topic string) error {

	err := s.deleteTopicKey(topic)
	if err != nil {
		log.Print("deleteTopic failed in removeTopic: ", err)
		return err
	}
	log.Printf("removed topic %s", topic)
	return nil
}

func (s *C2) newClientKey(id []byte) error {

	key := e4.RandomKey()

	// first send to the client, and only update locally afterwards
	payload, err := s.CreateAndProtectForID(e4.SetIDKey, nil, key, id)
	if err != nil {
		log.Print("CreateAndProtectForID failed in newClientKey: ", err)
		return err
	}
	err = s.sendToClient(id, payload)
	if err != nil {
		log.Print("sendToClient failed in newClientKey: ", err)
		return err
	}

	err = s.insertIDKey(id, key)
	if err != nil {
		log.Print("insertIDKey failed in newClientKey: ", err)
		return err
	}
	log.Printf("updated key for client %s", hex.EncodeToString(id))
	return nil
}
