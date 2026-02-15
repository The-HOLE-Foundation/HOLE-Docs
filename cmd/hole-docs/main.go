package main

import (
	"fmt"
	"os"

	"github.com/The-HOLE-Foundation/hole-docs/cmd/hole-docs/commands"
)

// Version is set during build via ldflags
var Version = "dev"

func main() {
	// Show version if requested
	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("hole-docs version %s\n", Version)
		os.Exit(0)
	}

	// Show help if no command provided
	if len(os.Args) < 2 {
		commands.ShowHelp(Version)
		os.Exit(0)
	}

	command := os.Args[1]

	// Route to appropriate command handler
	switch command {
	// PDF operations
	case "merge":
		commands.Merge(os.Args[2:])
	case "merge-dir":
		commands.MergeDir(os.Args[2:])
	case "split":
		commands.Split(os.Args[2:])
	case "optimize":
		commands.Optimize(os.Args[2:])
	case "validate":
		commands.Validate(os.Args[2:])
	case "info":
		commands.Info(os.Args[2:])

	// Rendering & conversion
	case "render":
		commands.Render(os.Args[2:])
	case "images-to-pdf":
		commands.ImagesToPDF(os.Args[2:])

	// Advanced features
	case "annotations":
		commands.Annotations(os.Args[2:])
	case "toc":
		commands.TableOfContents(os.Args[2:])

	// Document processing (AI-powered)
	case "upload":
		commands.Upload(os.Args[2:])
	case "search":
		commands.Search(os.Args[2:])
	case "list-docs":
		commands.ListDocs(os.Args[2:])

	// MCP server mode
	case "serve":
		commands.Serve(os.Args[2:])

	// Configuration
	case "config":
		commands.Config(os.Args[2:])

	// Help
	case "help", "-h", "--help":
		commands.ShowHelp(Version)

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		commands.ShowHelp(Version)
		os.Exit(1)
	}
}
