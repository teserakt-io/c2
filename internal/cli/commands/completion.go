package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
)

// CompletionCommand defines a custom Command to deal with auto completion
type CompletionCommand struct {
	cobraCmd *cobra.Command
	rootCmd  cli.Command
	flags    completionCommandFlags
}

type completionCommandFlags struct {
	IsZsh bool
}

var _ cli.Command = (*CompletionCommand)(nil)

// NewCompletionCommand returns the cobra command used to generate the autocompletion
func NewCompletionCommand(rootCommand cli.Command) *CompletionCommand {

	completionCmd := &CompletionCommand{
		rootCmd: rootCommand,
	}

	cobraCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

. <(c2cli completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(c2cli completion)`,
		RunE: completionCmd.run,
	}

	cobraCmd.Flags().BoolVar(
		&completionCmd.flags.IsZsh,
		"zsh",
		false,
		"Generate zsh completion script (default: bash)",
	)

	completionCmd.cobraCmd = cobraCmd

	return completionCmd
}

// CobraCmd returns the cobra command
func (c *CompletionCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *CompletionCommand) run(cmd *cobra.Command, args []string) error {
	if c.flags.IsZsh {
		c.rootCmd.CobraCmd().GenZshCompletion(os.Stdout)

		return nil
	}

	c.rootCmd.CobraCmd().GenBashCompletion(os.Stdout)

	return nil
}

// GenerateCustomCompletionFuncs returns the bash script snippets to use for custom autocompletion
func (c *CompletionCommand) GenerateCustomCompletionFuncs() string {
	var out string

	return out
}

func (c *CompletionCommand) generateCompletionFunc(funcName string, suggestions []string) string {
	return fmt.Sprintf(`
	%s()
	{
		COMPREPLY=( $(compgen -W "%s" -- "$cur") )
	}
	`,
		funcName,
		strings.Join(suggestions, " "),
	)
}
