package e4common

const (
	IdLen           = 32
	KeyLen          = 64
	TagLen          = 16 // no nonce in SIV, mac plays the role of nonce
	HashLen         = 32
	TimestampLen    = 8
	MaxTopicLen     = 512
	MaxSecondsDelay = 60 * 10
)

type Id []byte
type Topic string
type TopicHash []byte
type SymKey []byte

// TODO: add errors as var( ErrWTF = errors.New("..") )

type Command int

const (
	RemoveTopic Command = iota
	ResetTopics
	SetIdKey
	SetTopicKey
)

func (c *Command) ToByte() byte {
	switch *c {
	case RemoveTopic:
		return 0
	case ResetTopics:
		return 1
	case SetIdKey:
		return 2
	case SetTopicKey:
		return 3
	}
    return 255
}


func IsValidId(id []byte) bool {

	if len(id) != IdLen {
		return false
	}
	return true
}

func IsValidKey(key []byte) bool {

	if len(key) != KeyLen {
		return false
	}
	return true
}

func IsValidTopic(topic string) bool {

	if len(topic) > MaxTopicLen {
		return false
	}
	return true
}

func IsValidTopicHash(topichash []byte) bool {

	if len(topichash) != HashLen {
		return false
	}
	return true
}