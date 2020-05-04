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

package topics

import (
	"context"
	"fmt"
	"math"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type listCommand struct {
	cobraCmd        *cobra.Command
	flags           listCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type listCommandFlags struct {
	Offset int64
	Count  int64
}

var _ cli.Command = (*listCommand)(nil)

// NewListCommand creates a new command allowing to
// list existing topics
func NewListCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	listCmd := &listCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "List existing topics",
		RunE:  listCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().Int64Var(&listCmd.flags.Offset, "offset", 0, "The offset to start listing topics from")
	cobraCmd.Flags().Int64Var(&listCmd.flags.Count, "count", 0, "The maximum number of topics to return, values <= 0 means all")

	listCmd.cobraCmd = cobraCmd

	return listCmd
}

func (c *listCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *listCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	countResp, err := c2Client.CountTopics(ctx, &pb.CountTopicsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get topic count: %v", err)
	}

	totalCount := countResp.Count
	if c.flags.Count > 0 {
		// Will fetch as many as requested by user, up to maximum number of available topics
		totalCount = int64(math.Min(float64(totalCount), float64(c.flags.Count)))
	}

	currentOffset := c.flags.Offset
	for totalCount > 0 {
		count := int64(math.Min(float64(cli.MaxPageSize), float64(totalCount)))
		req := &pb.GetTopicsRequest{
			Count:  count,
			Offset: currentOffset,
		}

		resp, err := c2Client.GetTopics(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to fetch topics: %v", err)
		}

		currentOffset += count
		totalCount -= count
		for _, topic := range resp.Topics {
			c.CobraCmd().Println(topic)
		}
	}

	return nil
}
