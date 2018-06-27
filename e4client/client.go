package c2client

import (
	"errors"
	"log"
	"os"
	"encoding/gob"

	e4 "teserakt/e4common"
)


// structure saved to disk for persistent storage
type Client struct {
	id        []byte
	key       []byte
	topickeys map[string][]byte
	// slices []byte can't be map keys, converting to strings
	filePath  string
}

// TODO: init function, restore
// TODO: save client everytime it's changed

// creates a new client, generates random id of key if nil
func NewClient(id, key []byte, filePath string) *Client {
	if id == nil {
		id = e4.RandomId()
	}	
	if key == nil {
		key = e4.RandomKey()
	}
	topickeys := make(map[string][]byte)

	c := &Client{
		id: id,
		key: key,
		topickeys: topickeys,
		filePath: filePath,
	}
	
	return c
}

func LoadClient(filePath string) (*Client, error) {
	var c = new (Client)
	err := readGob(filePath, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) save() {
	err:= writeGob(c.filePath, c)
	if err != nil {
		log.Print("client save failed")	
	}
}

func writeGob(filePath string, object interface{}) error {
	file, err := os.Create(filePath)
	if err == nil {
		   encoder := gob.NewEncoder(file)
		   encoder.Encode(object)
	}
	file.Close()
	return err
}

func readGob(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err == nil {
		   decoder := gob.NewDecoder(file)
		   err = decoder.Decode(object)
	}
	file.Close()
	return err
}


// when se
func (c *Client) Protect(payload []byte, topic string) ([]byte, error) {
	topichash := string(e4.HashTopic(topic))
	if key, ok := c.topickeys[topichash]; ok {

		protected, err := e4.Protect(payload, key)
		if err != nil {
			return nil, err
		}
		return protected, nil
	}
	return nil, errors.New("topic key not found")
}

// when receiving with topic other than E4/c.id
func (c *Client) Unprotect(protected []byte, topic string) ([]byte, error) {
	topichash := string(e4.HashTopic(topic))
	if key, ok := c.topickeys[topichash]; ok {

		message, err := e4.Unprotect(protected, key)
		if err != nil {
			return nil, err
		}
		return message, nil
	}
	return nil, errors.New("topic key not found")
}

// when receiving with topic E4/c.id
func (c *Client) ProcessCommand(protected []byte) error {
	command, err := e4.Unprotect(protected, c.key)
	if err != nil {
		return err
	}

	cmd := e4.Command(command[0])
	
	switch cmd {

	case e4.RemoveTopic:
		if len(command) != e4.HashLen + 1 {
			return errors.New("invalid RemoveTopic argument")
		}
		return c.removeTopic(command[1:])

	case e4.ResetTopics:
		if len(command) != 1 {
			return errors.New("invalid ResetTopics argument")
		}
		return c.resetTopics()

	case e4.SetIdKey:
		if len(command) != e4.KeyLen + 1 {
			return errors.New("invalid SetIdKey argument")
		}
		return c.setIdKey(command[1:])
	
	case e4.SetTopicKey:
		if len(command) != e4.KeyLen + e4.HashLen + 1 {
			return errors.New("invalid SetTopicKey argument")
		}
		return c.setTopicKey(command[1:1+e4.HashLen], command[1+e4.HashLen:])
	
	default:
		return errors.New("invalid command")
	}
}

func (c *Client) removeTopic(topichash []byte) error {
	if !e4.IsValidTopicHash(topichash) {
		return errors.New("invalid topic hash")
	}
	delete(c.topickeys, string(topichash))

	return nil
}

func (c *Client) resetTopics() error {
	c.topickeys = make(map[string][]byte)
	return nil
}

func (c* Client) setIdKey(key []byte) error {
	c.key = key
	return nil
}

func (c *Client) setTopicKey(key, topichash []byte) error {
	c.topickeys[string(topichash)] = key
	return nil
}