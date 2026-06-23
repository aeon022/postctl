package main

import (
	_ "embed"

	"github.com/aeon022/postctl/cmd"
	"github.com/aeon022/postctl/internal/tui"
)

//go:embed README.md
var readmeContent string

func main() {
	tui.SetReadmeContent(readmeContent)
	cmd.Execute()
}
