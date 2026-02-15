package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/mcp"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// Serve starts the MCP server in HTTP or stdio mode
func Serve(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	httpMode := fs.Bool("http", false, "Start HTTP server (default mode)")
	stdioMode := fs.Bool("stdio", false, "Start stdio server for MCP clients (e.g., Claude Desktop)")
	port := fs.Int("port", 8080, "Port for HTTP mode")

	fs.Usage = func() {
		fmt.Println(`Usage: hole-docs serve [options]

Start MCP server for AI assistant integration.

The server provides 18+ tools for document processing that AI assistants
(like Claude) can use to help you work with PDFs and documents.

Modes:

  HTTP mode (default):
    Start a web server on specified port. AI assistants can call tools
    via HTTP requests.

    hole-docs serve --http --port 8080

  Stdio mode:
    Communicate via stdin/stdout for MCP clients like Claude Desktop.
    Configure in Claude Desktop's config.json:

    {
      "mcpServers": {
        "hole-docs": {
          "command": "/usr/local/bin/hole-docs",
          "args": ["serve", "--stdio"]
        }
      }
    }

Options:
  --http               Start HTTP server (default)
  --stdio              Start stdio server for Claude Desktop
  --port <port>        Port for HTTP mode (default: 8080)

Examples:
  # Start HTTP server on default port
  hole-docs serve

  # Start HTTP server on custom port
  hole-docs serve --port 3000

  # Start stdio mode for Claude Desktop
  hole-docs serve --stdio

Available Tools (18+):
  • merge_pdfs                 • split_pdf
  • optimize_pdf               • render_pdf_pages
  • images_to_pdf              • convert_image
  • extract_annotations        • add_toc_pdf
  • ingest_document            • search_documents
  • convert_docx_to_pdf        • and more...

See: https://github.com/The-HOLE-Foundation/hole-docs/blob/main/docs/api/MCP_TOOLS.md

Notes:
  • HTTP mode requires PORT environment variable or --port flag
  • Stdio mode requires MCP client (e.g., Claude Desktop)
  • All CLI commands are also available as MCP tools
  • Server runs until interrupted (Ctrl+C)
`)
	}

	fs.Parse(args)

	// Default to HTTP mode if neither specified
	if !*httpMode && !*stdioMode {
		*httpMode = true
	}

	// Load environment
	_ = godotenv.Load()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg := config.FromEnv()
	cfg.Port = *port

	logger.Info("Starting hole-docs MCP server",
		zap.String("mode", modeString(*httpMode, *stdioMode)),
		zap.Int("port", cfg.Port))

	if *stdioMode {
		// Stdio mode for MCP clients
		serveStdio(cfg, logger)
	} else {
		// HTTP mode for web access
		serveHTTP(cfg, logger)
	}
}

func modeString(http, stdio bool) string {
	if stdio {
		return "stdio"
	}
	return "http"
}

// serveHTTP starts the HTTP MCP server
func serveHTTP(cfg *config.Config, logger *zap.Logger) {
	// Initialize MCP server
	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize MCP server", zap.Error(err))
	}

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: server.Router(),
	}

	// Start server in goroutine
	go func() {
		logger.Info("MCP server listening",
			zap.String("addr", httpServer.Addr),
			zap.String("health", fmt.Sprintf("http://localhost:%d/health", cfg.Port)))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	fmt.Printf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║                     hole-docs MCP Server - HTTP Mode                         ║
╚══════════════════════════════════════════════════════════════════════════════╝

Server started successfully!

  Listening on:  http://localhost:%d
  Health check:  http://localhost:%d/health
  Tools list:    http://localhost:%d/tools

  Press Ctrl+C to stop

Available to AI assistants at: http://localhost:%d

`, cfg.Port, cfg.Port, cfg.Port, cfg.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	fmt.Println("\nShutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
	fmt.Println("Server stopped.")
}

// serveStdio starts the stdio MCP server
func serveStdio(cfg *config.Config, logger *zap.Logger) {
	// Initialize MCP server
	server, err := mcp.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize MCP server", zap.Error(err))
	}

	logger.Info("MCP server starting in stdio mode")

	// TODO: Implement stdio protocol
	// For now, show that stdio mode is recognized
	fmt.Fprintln(os.Stderr, "hole-docs MCP Server - Stdio Mode")
	fmt.Fprintln(os.Stderr, "Waiting for MCP protocol messages on stdin...")

	// Placeholder - actual implementation would handle MCP protocol
	// over stdin/stdout
	logger.Warn("Stdio mode not yet fully implemented",
		zap.String("status", "coming_soon"))

	fmt.Fprintln(os.Stderr, "⚠️  Stdio mode coming soon!")
	fmt.Fprintln(os.Stderr, "For now, use: hole-docs serve --http")

	_ = server // Use server to avoid unused variable error
}
