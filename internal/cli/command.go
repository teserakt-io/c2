package cli

import "github.com/spf13/cobra"

// Command defines a cli Command
type Command interface {
	CobraCmd() *cobra.Command
}
