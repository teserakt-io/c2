package commands

import (
	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/internal/cli/commands/clients"
	"github.com/teserakt-io/c2/internal/cli/commands/topics"
)

type rootCommandFlags struct {
	Endpoint string
	Cert     string
}

type rootCommand struct {
	cobraCmd *cobra.Command
	flags    rootCommandFlags
}

var _ cli.Command = (*rootCommand)(nil)

// NewRootCommand creates and configure a new cli root command
func NewRootCommand(c2ClientFactory cli.APIClientFactory, version string) cli.Command {
	rootCmd := &rootCommand{}

	clientCommand := clients.NewRootCommand(c2ClientFactory)
	topicCommand := topics.NewRootCommand(c2ClientFactory)

	countCommand := NewCountCommand(c2ClientFactory)
	attachCommand := NewAttachCommand(c2ClientFactory)
	detachCommand := NewDetachCommand(c2ClientFactory)

	eventsCommand := NewEventsCommand(c2ClientFactory)

	// TODO: disabled for now as it need a fair bit of polish before being usable
	//interactiveCmd := NewInteractiveCommand(rootCmd, version)
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

		countCommand.CobraCmd(),
		attachCommand.CobraCmd(),
		detachCommand.CobraCmd(),

		eventsCommand.CobraCmd(),
		// TODO: disabled for now as it need a fair bit of polish before being usable
		//interactiveCmd.CobraCmd(),

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
