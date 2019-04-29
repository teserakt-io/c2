package models

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
