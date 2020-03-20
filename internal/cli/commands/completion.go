// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"os"

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
