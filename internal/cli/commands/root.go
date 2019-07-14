package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/internal/cli/commands/clients"
	"gitlab.com/teserakt/c2/internal/cli/commands/topics"
)

type rootCommandFlags struct {
	Endpoint    string
	Cert        string
	Interactive bool
}

type rootCommand struct {
	cobraCmd *cobra.Command
	flags    rootCommandFlags
}

var _ cli.Command = &rootCommand{}

// NewRootCommand creates and configure a new cli root command
func NewRootCommand(c2ClientFactory cli.APIClientFactory, version string) cli.Command {
	rootCmd := &rootCommand{}

	clientCommand := clients.NewRootCommand(c2ClientFactory)
	topicCommand := topics.NewRootCommand(c2ClientFactory)

	completionCmd := NewCompletionCommand(rootCmd)

	cobraCmd := &cobra.Command{
		Use:                    "c2cli",
		BashCompletionFunction: completionCmd.GenerateCustomCompletionFuncs(),
		Version:                version,
		SilenceUsage:           true,
		SilenceErrors:          true,
		RunE:                   rootCmd.run,
	}

	cobraCmd.Flags().BoolVarP(&rootCmd.flags.Interactive, "interactive", "i", false, "enter c2cli interactive mode")

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

func (c *rootCommand) run(cmd *cobra.Command, args []string) error {
	if !c.flags.Interactive {
		return nil
	}

	fmt.Println("c2cli interactive console")
	fmt.Println("enter 'bye' to quit")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("c2cli> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}

		// Remove \n from input
		text = text[:len(text)-1]

		if text == "bye" {
			fmt.Println("goodbye")
			return nil
		}

		c.cobraCmd.SetArgs(strings.Split(text, " "))
		if err := c.cobraCmd.Execute(); err != nil {
			fmt.Printf("error running command: %v\n", err)
		}
	}
}
