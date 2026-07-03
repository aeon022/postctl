package cmd

import (
	"github.com/aeon022/postctl/internal/mcpserver"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio) — exposes post tools to AI",
	Long: `Starts a Model Context Protocol server over stdio.
Connect from Claude Desktop or any MCP-compatible client.

Tools exposed:
  list_posts       List posts (filter by platform/status/campaign)
  get_post         Get a post by ID
  create_post      Create a new draft or scheduled post
  publish_post     Publish a post immediately
  schedule_post    Update a post's scheduled time
  list_campaigns   List all campaigns with status breakdown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpserver.Serve()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
