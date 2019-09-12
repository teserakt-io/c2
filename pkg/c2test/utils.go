package c2test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"strings"

	e4crypto "github.com/teserakt-io/e4go/crypto"
)

// TestClient provides a representation of an id-key pair.
type TestClient struct {
	Name string
	ID   []byte
	Key  []byte
}

// TestTopic provides a representation of the topic-key pair.
type TestTopic struct {
	TopicName string
	Key       []byte
}

// TestResult reports a test outcome for nice display
type TestResult struct {
	Name     string
	Result   bool
	Critical bool
	Error    error
}

// Print will write the TestResult to given writer
func (result TestResult) Print(w io.Writer) {
	status := "fail"
	if result.Result {
		status = "pass"
	}

	pad := strings.Repeat(" ", 60-len(result.Name))
	fmt.Fprintf(w, "  - Test \"%s\" %s - %s\n", result.Name, pad, status)
	if result.Error != nil {
		fmt.Fprintf(w, "    Error: %v\n", result.Error)
	}
}

// GetRandomDBName returns a random database filepath to be used with sqlite driver
func GetRandomDBName() string {
	bytes := [16]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		panic(err)
	}
	dbCandidate := base64.StdEncoding.EncodeToString(bytes[:])
	dbCleaned1 := strings.Replace(dbCandidate, "+", "", -1)
	dbCleaned2 := strings.Replace(dbCleaned1, "/", "", -1)
	dbCleaned3 := strings.Replace(dbCleaned2, "=", "", -1)

	dbPath := fmt.Sprintf("/tmp/e4c2_unittest_%s.sqlite", dbCleaned3)
	return dbPath
}

// CheckC2Online is a quick function to wait until the C2 is online
// It does this by defining the C2 online as when both GRPC and HTTP
// APIs are up and accepting connections.
// TODO: We might want to formally define some way to deduce if the C2 is
// loaded, for example by a GRPC request
// So that all services and any backend initialization is ready.
func CheckC2Online(addr string, grpcPort int, httpPort int) bool {
	connGrpc, err := net.Dial("tcp", fmt.Sprintf("%s:%d", addr, grpcPort))
	if err != nil {
		return false
	}
	connGrpc.Close()

	connHTTP, err := net.Dial("tcp", fmt.Sprintf("%s:%d", addr, httpPort))
	if err != nil {
		return false
	}
	connHTTP.Close()

	return true
}

// NewTestClient generates a new TestClient
func NewTestClient() (*TestClient, error) {
	t := &TestClient{}
	name, err := GenerateName()
	if err != nil {
		return nil, err
	}
	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	t.Name = name
	t.ID = e4crypto.HashIDAlias(name)
	t.Key = key
	return t, nil
}

// NewTestClientWithoutName generates a new TestClient without name
func NewTestClientWithoutName() (*TestClient, error) {
	t := &TestClient{}
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}
	key, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	t.Name = ""
	t.ID = id
	t.Key = key
	return t, nil
}

// NewTestTopic generates a new TestTopic
func NewTestTopic(topickeygen bool) (*TestTopic, error) {
	t := &TestTopic{}
	topic, err := GenerateTopic()
	if err != nil {
		return nil, err
	}
	if topickeygen {
		key, err := GenerateKey()
		if err != nil {
			return nil, err
		}
		t.Key = key
	}
	t.TopicName = topic
	return t, nil
}

// GenerateName generates a random name beginning clientname-%s
// it is used to make random-looking names for devices for test purposes.
func GenerateName() (string, error) {
	somebytes := [e4crypto.IDLen]byte{}
	_, err := rand.Read(somebytes[:])
	if err != nil {
		return "", err
	}
	encodedbytes := hex.EncodeToString(somebytes[:])
	clientname := fmt.Sprintf("clientname%s", encodedbytes)
	return clientname, nil
}

// GenerateID generates a random ID that is e4.IDLen bytes
// in length, using a CSPRNG
func GenerateID() ([]byte, error) {
	idBytes := [e4crypto.IDLen]byte{}
	_, err := rand.Read(idBytes[:])
	if err != nil {
		return nil, err
	}
	return idBytes[:], nil
}

// GenerateKey generates a random key that is e4.KeyLen bytes
// in length, using a CSPRNG
func GenerateKey() ([]byte, error) {
	keybytes := [e4crypto.KeyLen]byte{}
	_, err := rand.Read(keybytes[:])
	if err != nil {
		return nil, err
	}
	return keybytes[:], nil
}

// GenerateTopic generates a random topic
func GenerateTopic() (string, error) {
	bytes := [28]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", err
	}
	tCandidate := base64.StdEncoding.EncodeToString(bytes[:])
	tCleaned1 := strings.Replace(tCandidate, "+", "", -1)
	tCleaned2 := strings.Replace(tCleaned1, "/", "", -1)
	tCleaned3 := strings.Replace(tCleaned2, "=", "", -1)

	len := int(math.Min(float64(len(tCleaned3)), 32))

	return tCleaned3[0:len], nil
}
