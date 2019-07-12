package commands

import (
	"github.com/spf13/cobra"
	"gitlab.com/teserakt/c2/internal/cli"
)

// Command defines a cli Command
type Command interface {
	CobraCmd() *cobra.Command
}

type rootCommandFlags struct {
	Endpoint string
	Cert     string
}

type rootCommand struct {
	cobraCmd *cobra.Command
	flags    rootCommandFlags
}

var _ Command = &rootCommand{}

// NewRootCommand creates and configure a new cli root command
func NewRootCommand(c2ClientFactory cli.APIClientFactory, version string) Command {
	rootCmd := &rootCommand{}

	clientCommand := NewClientCommand(c2ClientFactory)
	topicCommand := NewTopicCommand(c2ClientFactory)

	completionCmd := NewCompletionCommand(rootCmd)

	cobraCmd := &cobra.Command{
		Use:                    "c2cli",
		BashCompletionFunction: completionCmd.GenerateCustomCompletionFuncs(),
		Version:                version,
		SilenceUsage:           true,
		SilenceErrors:          true,
	}

	cobraCmd.PersistentFlags().StringVarP(
		&rootCmd.flags.Endpoint,
		cli.EndpointFlag,
		"e",
		"127.0.0.1:5555", "url to the c2 grpc api",
	)

	cobraCmd.PersistentFlags().StringVarP(
		&rootCmd.flags.Cert,
		cli.CertFlag,
		"c",
		"configs/c2-cert.pem", "path to the c2 grpc api certificate",
	)

	cobraCmd.AddCommand(
		clientCommand.CobraCmd(),
		topicCommand.CobraCmd(),

		// Autocompletion script generation command
		completionCmd.CobraCmd(),
	)

	cobraCmd.SetVersionTemplate(`{{printf "%s" .Version}}`)

	rootCmd.cobraCmd = cobraCmd

	return rootCmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
