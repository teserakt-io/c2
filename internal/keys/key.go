package keys

import (
	"github.com/teserakt-io/c2/internal/commands"
)

type E4Key interface {
	// ProtectMessage encrypt given payload using the topicKey
	// and returns the protected cipher, or an error
	ProtectMessage(payload []byte, topicKey []byte) ([]byte, error)
	// ProtectCommand encrypt the given command using the key material private key
	// and returns the protected command, or an error
	ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error)

	// Unprotect* are not needed now by C2, but we can still have them ?

	// UnprotectMessage decrypt the given cipher using the topicKey
	// and returns the clear payload, or an error
	UnprotectMessage(protected []byte, topicKey []byte) ([]byte, error)
	// UnprotectCommand decrypt the given protected command using the key material private key
	// and returns the command, or an error
	UnprotectCommand(protected []byte) (commands.Command, error)
}
