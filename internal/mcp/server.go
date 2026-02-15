package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/The-HOLE-Foundation/hole-docs/internal/annotations"
	"github.com/The-HOLE-Foundation/hole-docs/internal/config"
	"github.com/The-HOLE-Foundation/hole-docs/internal/devonthink"
	"github.com/The-HOLE-Foundation/hole-docs/internal/images"
	"github.com/The-HOLE-Foundation/hole-docs/internal/models"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pdf"
	"github.com/The-HOLE-Foundation/hole-docs/internal/pipeline"
	"github.com/The-HOLE-Foundation/hole-docs/internal/storage"
	"github.com/The-HOLE-Foundation/hole-docs/internal/word"
	"github.com/google/uuid"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

// Server represents the MCP server
type Server struct {
	config              *config.Config
	logger              *zap.Logger
	tools               map[string]*ToolHandler
	cache               *storage.Cache
	startTime           time.Time
	mu                  sync.RWMutex
	pdfProcessor        *pdf.Processor
	imgProcessor        *images.Processor
	wordProcessor       *word.Processor
	dtResolver          *devonthink.Resolver
	processingOrchestrator *pipeline.ProcessingOrchestrator
}

// ToolHandler is a function that handles tool invocations
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// NewServer creates a new MCP server instance
func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Create cache
	cache := storage.NewCache(cfg.TempDir)

	// Create PDF processor
	pdfProc := pdf.NewProcessor(cache, logger)

	// Create image processor
	imgProc := images.NewProcessor(cache, logger)

	// Create Word processor
	wordProc := word.NewProcessor(cache, logger)

	// Create DEVONthink resolver
	dtResolver := devonthink.NewResolver(logger)

	// Create R2 uploader if credentials are available
	var r2Uploader *storage.R2Uploader
	if cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" && cfg.R2Endpoint != "" {
		uploader, err := storage.NewR2Uploader(
			cfg.R2AccessKeyID,
			cfg.R2SecretAccessKey,
			cfg.R2Endpoint,
			cfg.R2BucketName,
			logger,
		)
		if err != nil {
			logger.Warn("Failed to initialize R2 uploader", zap.Error(err))
		} else {
			r2Uploader = uploader
		}
	}

	// Create processing orchestrator
	orchestrator := pipeline.NewProcessingOrchestrator(r2Uploader, nil, nil, nil, nil, logger)

	server := &Server{
		config:              cfg,
		logger:              logger,
		tools:               make(map[string]*ToolHandler),
		cache:               cache,
		startTime:           time.Now(),
		pdfProcessor:        pdfProc,
		imgProcessor:        imgProc,
		wordProcessor:       wordProc,
		dtResolver:          dtResolver,
		processingOrchestrator: orchestrator,
	}

	// Register tools
	server.registerTools()

	return server, nil
}

// registerTools registers all available tools
func (s *Server) registerTools() {
	// PDF tools
	s.registerTool("merge_pdfs", "Merge multiple PDF files using qpdf (fast, lossless - STANDARD METHOD)", s.handleMergePDFs)
	s.registerTool("merge_pdfs_qpdf", "Merge PDFs using qpdf with advanced options", s.handleMergePDFsQPDF)
	s.registerTool("merge_directory_qpdf", "Merge all PDFs in a directory using qpdf", s.handleMergeDirectoryQPDF)
	s.registerTool("validate_pdf_qpdf", "Validate a PDF file using qpdf", s.handleValidatePDFQPDF)
	s.registerTool("get_pdf_info_qpdf", "Get detailed information about a PDF using qpdf", s.handleGetPDFInfoQPDF)
	s.registerTool("split_pdf", "Split a PDF file into individual pages or page ranges", s.handleSplitPDF)
	s.registerTool("optimize_pdf", "Optimize a PDF file for smaller file size", s.handleOptimizePDF)
	s.registerTool("pdf_to_images", "Export PDF pages as individual image files", s.handlePDFToImages)
	s.registerTool("render_pdf_pages", "Render PDF pages to PNG/JPEG images (lossless quality) - use 'processor' param: 'ghostscript' (default) or 'mupdf'", s.handleRenderPDFPages)
	s.registerTool("add_toc_pdf", "Add table of contents to a PDF", s.handleAddTOC)
	s.registerTool("extract_annotations", "Extract reviewer annotations from PDF for LLM revision workflow", s.handleExtractAnnotations)

	// Image tools
	s.registerTool("images_to_pdf", "Convert image files to a PDF document", s.handleImagesToPDF)
	s.registerTool("merge_images", "Merge multiple images into a PDF or single image", s.handleMergeImages)
	s.registerTool("convert_image", "Convert image format (PNG, JPEG, WebP, etc.)", s.handleConvertImage)

	// Word document tools
	s.registerTool("convert_docx_to_pdf", "Convert Word document (.docx) to PDF", s.handleConvertDocxToPdf)
	s.registerTool("extract_text_from_docx", "Extract text from Word document", s.handleExtractTextFromDocx)
	s.registerTool("get_docx_metadata", "Get Word document metadata", s.handleGetDocxMetadata)

	// Document processing tools
	s.registerTool("ingest_document", "Ingest and process a document (PDF, DOCX, or image) through the pipeline", s.handleIngestDocument)
	s.registerTool("ingest_batch", "Process multiple documents in batch (sequential - MVP version)", s.handleIngestBatch)

	// Server tools
	s.registerTool("list_tools", "List all available tools", s.handleListTools)
	s.registerTool("health_check", "Check server health and status", s.handleHealthCheck)
	s.registerTool("search_devonthink", "Search DEVONthink database and get document links", s.handleSearchDEVONthink)
}

// registerTool registers a single tool
func (s *Server) registerTool(name string, description string, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[name] = &handler
	s.logger.Info("Tool registered", zap.String("tool", name))
}

// Router returns the HTTP router
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHTTPHealthCheck)

	// Server info endpoint
	mux.HandleFunc("/info", s.handleHTTPServerInfo)

	// Tool invocation endpoint
	mux.HandleFunc("/invoke", s.handleHTTPInvoke)

	// Tool list endpoint
	mux.HandleFunc("/tools", s.handleHTTPListTools)

	// File upload endpoint
	mux.HandleFunc("/upload", s.handleHTTPUpload)

	// Apply CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
	})

	return c.Handler(mux)
}

// handleHTTPHealthCheck handles health check requests
func (s *Server) handleHTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime).Seconds()
	health := models.HealthCheck{
		Status:  "healthy",
		Version: "0.1.0",
		Uptime:  int64(uptime),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleHTTPServerInfo handles server info requests
func (s *Server) handleHTTPServerInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]models.Tool, 0, len(s.tools))
	for name := range s.tools {
		tools = append(tools, models.Tool{
			Name:        name,
			Description: "Tool description",
		})
	}

	info := models.ServerInfo{
		Name:        "GoDocs MCP Server",
		Version:     "0.1.0",
		Description: "PDF and image processing MCP server in Go",
		Tools:       tools,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleHTTPListTools handles tool listing requests
func (s *Server) handleHTTPListTools(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make(map[string]string)
	for name := range s.tools {
		tools[name] = "Available"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
}

// handleHTTPInvoke handles tool invocation requests
func (s *Server) handleHTTPInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ToolCall
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	handler, exists := s.tools[req.ToolName]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Tool not found: %s", req.ToolName), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	result, err := (*handler)(ctx, req.Args)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleHTTPUpload handles file uploads
func (s *Server) handleHTTPUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(s.config.MaxUploadSize); err != nil {
		http.Error(w, fmt.Sprintf("Upload too large: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("File upload failed: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate unique ID for uploaded file
	fileID := uuid.New().String()

	// Save file to cache
	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	cachedPath, err := s.cache.SaveFile(fileID, fileData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"file_id":  fileID,
		"filename": header.Filename,
		"size":     len(fileData),
		"path":     cachedPath,
	})
}

// Tool handlers

func (s *Server) handleMergePDFs(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Merging PDFs with qpdf (standard method)", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPaths, ok := params["input_paths"].([]interface{})
	if !ok || len(inputPaths) == 0 {
		return nil, fmt.Errorf("input_paths is required and must be a list")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Convert []interface{} to []string
	paths := make([]string, len(inputPaths))
	for i, p := range inputPaths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("all input_paths must be strings")
		}
		paths[i] = path
	}

	// Resolve DEVONthink links to file paths
	resolvedPaths, err := s.dtResolver.ResolveLinksToPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DEVONthink links: %w", err)
	}

	// Parse optional parameters
	opts := &pdf.QPDFMergeOptions{}
	if removeDups, ok := params["remove_duplicates"].(bool); ok {
		opts.RemoveDuplicates = removeDups
	}
	if linearize, ok := params["linearize"].(bool); ok {
		opts.Linearize = linearize
	}

	// Check if legacy pdfcpu method is explicitly requested
	useLegacy := false
	if legacy, ok := params["use_pdfcpu"].(bool); ok {
		useLegacy = legacy
	}

	// Use qpdf by default, or pdfcpu if explicitly requested
	if useLegacy {
		s.logger.Info("Using legacy pdfcpu method (explicitly requested)")
		if err := s.pdfProcessor.MergePDFs(resolvedPaths, outputPath); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status": "success",
			"output_path": outputPath,
			"method": "pdfcpu (legacy)",
			"resolved_paths": resolvedPaths,
		}, nil
	}

	// Standard: Use qpdf for fast, lossless merge
	if err := s.pdfProcessor.MergePDFsWithQPDF(resolvedPaths, outputPath, opts); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"method": "qpdf (standard)",
		"remove_duplicates": opts.RemoveDuplicates,
		"linearize": opts.Linearize,
		"resolved_paths": resolvedPaths,
	}, nil
}

func (s *Server) handleSplitPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Splitting PDF", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputDir, ok := params["output_dir"].(string)
	if !ok || outputDir == "" {
		return nil, fmt.Errorf("output_dir is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	pageRanges, ok := params["page_ranges"].([]interface{})
	if !ok || len(pageRanges) == 0 {
		return nil, fmt.Errorf("page_ranges is required and must be a list (e.g., ['1-5', '10', '20-25'])")
	}

	// Convert []interface{} to []string
	ranges := make([]string, len(pageRanges))
	for i, r := range pageRanges {
		rangeStr, ok := r.(string)
		if !ok {
			return nil, fmt.Errorf("all page_ranges must be strings")
		}
		ranges[i] = rangeStr
	}

	// Call processor
	if err := s.pdfProcessor.SplitPDF(inputPath, outputDir, ranges); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_dir": outputDir,
	}, nil
}

func (s *Server) handleOptimizePDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Optimizing PDF", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	// DPI and Quality parameters
	// Support both preset names ("72dpi", "172dpi", etc.) and custom values
	dpi := 172       // Default to 172 DPI (optimal for legal documents)
	quality := 78    // Default quality
	presetName := "" // For tracking which preset was used

	// Check for preset name
	if p, ok := params["preset"].(string); ok && p != "" {
		if preset, exists := pdf.PredefinedPresets[p]; exists {
			dpi = preset.DPI
			quality = preset.JpegQuality
			presetName = preset.Name
			s.logger.Info("Using preset", zap.String("preset", presetName))
		}
	}

	// Allow custom DPI override (only if no preset specified)
	if presetName == "" {
		if d, ok := params["dpi"].(float64); ok {
			dpi = int(d)
		}

		// Allow custom quality override
		if q, ok := params["quality"].(float64); ok {
			quality = int(q)
		}
	}

	// Check if advanced mode requested (use Ghostscript)
	useAdvanced := true // Default to Ghostscript now
	if u, ok := params["advanced"].(bool); ok {
		useAdvanced = u
	}

	var err error
	if useAdvanced {
		// Use Ghostscript-based optimization with DPI control
		err = s.pdfProcessor.OptimizePDFAdvanced(inputPath, outputPath, dpi, quality)
	} else {
		// Use basic pdfcpu optimization (legacy fallback)
		err = s.pdfProcessor.OptimizePDF(inputPath, outputPath, "medium")
	}

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"status":      "success",
		"output_path": outputPath,
		"dpi":         dpi,
		"quality":     quality,
	}

	if presetName != "" {
		result["preset"] = presetName
	}

	return result, nil
}

func (s *Server) handlePDFToImages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Converting PDF to images", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputDir, ok := params["output_dir"].(string)
	if !ok || outputDir == "" {
		return nil, fmt.Errorf("output_dir is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	format := "png"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	dpi := 150
	if d, ok := params["dpi"].(float64); ok {
		dpi = int(d)
	}

	// Call processor
	if err := s.pdfProcessor.ExportToImages(inputPath, outputDir, format, dpi); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_dir": outputDir,
		"format": format,
	}, nil
}

func (s *Server) handleRenderPDFPages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Rendering PDF pages to images", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputDir, ok := params["output_dir"].(string)
	if !ok || outputDir == "" {
		return nil, fmt.Errorf("output_dir is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	format := "png"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	dpi := 300 // Default to high quality for lossless
	if d, ok := params["dpi"].(float64); ok {
		dpi = int(d)
	}

	// Extract processor parameter: "ghostscript" (default), "mupdf", or "mu"
	processor := "ghostscript"
	if p, ok := params["processor"].(string); ok && p != "" {
		processor = p
	}

	s.logger.Info("Using processor for rendering",
		zap.String("processor", processor),
		zap.String("format", format),
		zap.Int("dpi", dpi))

	// Call processor with specified renderer
	if err := s.pdfProcessor.RenderPagesToImagesWithProcessor(inputPath, outputDir, format, dpi, processor); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_dir": outputDir,
		"format": format,
		"dpi": dpi,
		"processor": processor,
		"resolved_path": inputPath,
	}, nil
}

func (s *Server) handleAddTOC(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Adding TOC to PDF", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	entriesRaw, ok := params["entries"].([]interface{})
	if !ok || len(entriesRaw) == 0 {
		return nil, fmt.Errorf("entries is required and must be a list of {title, page, level}")
	}

	// Convert []interface{} to []TOCEntry
	entries := make([]pdf.TOCEntry, len(entriesRaw))
	for i, e := range entriesRaw {
		entryMap, ok := e.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each entry must be an object")
		}

		title, ok := entryMap["title"].(string)
		if !ok {
			return nil, fmt.Errorf("entry title must be a string")
		}

		page, ok := entryMap["page"].(float64)
		if !ok {
			return nil, fmt.Errorf("entry page must be a number")
		}

		level := 0
		if l, ok := entryMap["level"].(float64); ok {
			level = int(l)
		}

		entries[i] = pdf.TOCEntry{
			Title: title,
			Page:  int(page),
			Level: level,
		}
	}

	// Call processor
	if err := s.pdfProcessor.AddTableOfContents(inputPath, outputPath, entries); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"entries_added": len(entries),
	}, nil
}

func (s *Server) handleExtractAnnotations(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Extracting annotations from PDF", zap.Any("params", params))

	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}
	inputPath = resolvedInputPath

	if _, err := os.Stat(inputPath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Build extraction options
	opts := annotations.DefaultOptions()
	if includeLinks, ok := params["include_links"].(bool); ok {
		opts.IncludeLinks = includeLinks
	}
	if enableOCR, ok := params["enable_ocr"].(bool); ok {
		opts.RunOCR = enableOCR
	}
	if pages, ok := params["selected_pages"].([]interface{}); ok {
		for _, p := range pages {
			if ps, ok := p.(string); ok {
				opts.SelectedPages = append(opts.SelectedPages, ps)
			}
		}
	}

	// Path A: structured extraction via pdfcpu
	extractor := annotations.NewPDFExtractor(s.logger)
	annotSet, err := extractor.ExtractFromFile(inputPath, opts)
	if err != nil {
		return nil, fmt.Errorf("annotation extraction failed: %w", err)
	}

	// Path B: optional render+OCR
	if opts.RunOCR {
		ocrExtractor := annotations.NewOCRExtractor(s.logger)
		ocrText, err := ocrExtractor.RenderAndOCR(inputPath)
		if err != nil {
			s.logger.Warn("OCR extraction failed, continuing without OCR text", zap.Error(err))
			annotSet.Errors = append(annotSet.Errors, fmt.Sprintf("OCR: %v", err))
		} else {
			annotSet.OCRText = ocrText
		}
	}

	// Format output
	format := extractParamString(params, "format", "llm")
	var output interface{}
	switch format {
	case "json":
		output = annotSet
	case "markdown":
		output = map[string]interface{}{
			"status":   "success",
			"format":   "markdown",
			"output":   annotations.FormatAsMarkdown(annotSet),
			"summary":  annotSet.Summary,
		}
	case "chat":
		output = map[string]interface{}{
			"status":   "success",
			"format":   "chat",
			"output":   annotations.FormatAsChatContext(annotSet),
			"summary":  annotSet.Summary,
		}
	default: // "llm"
		output = map[string]interface{}{
			"status":   "success",
			"format":   "llm",
			"output":   annotations.FormatForLLM(annotSet),
			"summary":  annotSet.Summary,
		}
	}

	return output, nil
}

// qpdf-based handlers for fast, lossless PDF operations

func (s *Server) handleMergePDFsQPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Merging PDFs with qpdf", zap.Any("params", params))

	// Extract parameters
	inputPaths, ok := params["input_paths"].([]interface{})
	if !ok || len(inputPaths) == 0 {
		return nil, fmt.Errorf("input_paths is required and must be a list")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Convert []interface{} to []string
	paths := make([]string, len(inputPaths))
	for i, p := range inputPaths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("all input_paths must be strings")
		}
		paths[i] = path
	}

	// Resolve DEVONthink links to file paths
	resolvedPaths, err := s.dtResolver.ResolveLinksToPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DEVONthink links: %w", err)
	}

	// Parse optional parameters
	opts := &pdf.QPDFMergeOptions{}
	if removeDups, ok := params["remove_duplicates"].(bool); ok {
		opts.RemoveDuplicates = removeDups
	}
	if linearize, ok := params["linearize"].(bool); ok {
		opts.Linearize = linearize
	}

	// Call processor
	if err := s.pdfProcessor.MergePDFsWithQPDF(resolvedPaths, outputPath, opts); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"input_count": len(resolvedPaths),
		"remove_duplicates": opts.RemoveDuplicates,
		"linearize": opts.Linearize,
		"resolved_paths": resolvedPaths,
	}, nil
}

func (s *Server) handleMergeDirectoryQPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Merging PDFs from directory with qpdf", zap.Any("params", params))

	// Extract parameters
	inputDir, ok := params["input_dir"].(string)
	if !ok || inputDir == "" {
		return nil, fmt.Errorf("input_dir is required")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Optional parameters
	recursive := false
	if r, ok := params["recursive"].(bool); ok {
		recursive = r
	}

	opts := &pdf.QPDFMergeOptions{}
	if removeDups, ok := params["remove_duplicates"].(bool); ok {
		opts.RemoveDuplicates = removeDups
	}
	if linearize, ok := params["linearize"].(bool); ok {
		opts.Linearize = linearize
	}

	// Call processor
	if err := s.pdfProcessor.MergePDFDirectories(inputDir, outputPath, recursive, opts); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"input_dir": inputDir,
		"recursive": recursive,
		"remove_duplicates": opts.RemoveDuplicates,
		"linearize": opts.Linearize,
	}, nil
}

func (s *Server) handleValidatePDFQPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Validating PDF with qpdf", zap.Any("params", params))

	// Extract parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedPath, err := s.dtResolver.ResolveLinkToPath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Call processor
	valid, err := s.pdfProcessor.ValidatePDFWithQPDF(resolvedPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"file_path": resolvedPath,
		"valid": valid,
	}, nil
}

func (s *Server) handleGetPDFInfoQPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Getting PDF info with qpdf", zap.Any("params", params))

	// Extract parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedPath, err := s.dtResolver.ResolveLinkToPath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Call processor
	info, err := s.pdfProcessor.GetPDFInfoWithQPDF(resolvedPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"file_path": info.FilePath,
		"page_count": info.PageCount,
		"valid": info.Valid,
		"encrypted": info.Encrypted,
		"linear_maps": info.LinearMaps,
		"error": info.Error,
	}, nil
}

func (s *Server) handleImagesToPDF(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Converting images to PDF", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPaths, ok := params["input_paths"].([]interface{})
	if !ok || len(inputPaths) == 0 {
		return nil, fmt.Errorf("input_paths is required and must be a list")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Convert []interface{} to []string
	paths := make([]string, len(inputPaths))
	for i, p := range inputPaths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("all input_paths must be strings")
		}
		paths[i] = path
	}

	// Resolve DEVONthink links to file paths
	resolvedPaths, err := s.dtResolver.ResolveLinksToPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DEVONthink links: %w", err)
	}

	// Call processor
	if err := s.imgProcessor.ConvertToPDF(resolvedPaths, outputPath); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"images_converted": len(resolvedPaths),
		"resolved_paths": resolvedPaths,
	}, nil
}

func (s *Server) handleMergeImages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Merging images", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPaths, ok := params["input_paths"].([]interface{})
	if !ok || len(inputPaths) == 0 {
		return nil, fmt.Errorf("input_paths is required and must be a list")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	format := "pdf"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	// Convert []interface{} to []string
	paths := make([]string, len(inputPaths))
	for i, p := range inputPaths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("all input_paths must be strings")
		}
		paths[i] = path
	}

	// Resolve DEVONthink links to file paths
	resolvedPaths, err := s.dtResolver.ResolveLinksToPaths(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DEVONthink links: %w", err)
	}

	// Call processor
	if err := s.imgProcessor.MergeImages(resolvedPaths, outputPath, format); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"format": format,
		"images_merged": len(resolvedPaths),
		"resolved_paths": resolvedPaths,
	}, nil
}

func (s *Server) handleConvertImage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Converting image", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	targetFormat, ok := params["target_format"].(string)
	if !ok || targetFormat == "" {
		return nil, fmt.Errorf("target_format is required (png, jpeg, webp)")
	}

	quality := 85
	if q, ok := params["quality"].(float64); ok {
		quality = int(q)
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, err := s.dtResolver.ResolveLinkToPath(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", err)
	}

	// Call processor
	if err := s.imgProcessor.ConvertFormat(resolvedInputPath, outputPath, targetFormat, quality); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status": "success",
		"output_path": outputPath,
		"format": targetFormat,
		"resolved_path": resolvedInputPath,
	}, nil
}

func (s *Server) handleListTools(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tools := make([]string, 0, len(s.tools))
	for name := range s.tools {
		tools = append(tools, name)
	}
	return map[string]interface{}{"tools": tools}, nil
}

func (s *Server) handleHealthCheck(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	uptime := time.Since(s.startTime).Seconds()
	return models.HealthCheck{
		Status:  "healthy",
		Version: "0.1.0",
		Uptime:  int64(uptime),
	}, nil
}

func (s *Server) handleSearchDEVONthink(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Searching DEVONthink", zap.Any("params", params))

	// Extract search query
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Optional limit
	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Search DEVONthink
	results, err := s.dtResolver.SearchDocuments(query, limit)
	if err != nil {
		return nil, err
	}

	// Format results for JSON response
	formattedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		formattedResults[i] = map[string]interface{}{
			"uuid": result.UUID,
			"name": result.Name,
			"path": result.Path,
			"link": result.Link,
		}
	}

	return map[string]interface{}{
		"status": "success",
		"count":  len(results),
		"query":  query,
		"results": formattedResults,
	}, nil
}

// Word document handlers

func (s *Server) handleConvertDocxToPdf(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Converting Word to PDF", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	outputPath, ok := params["output_path"].(string)
	if !ok || outputPath == "" {
		return nil, fmt.Errorf("output_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}

	// Call processor
	if err := s.wordProcessor.ConvertToPDF(resolvedInputPath, outputPath); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":        "success",
		"output_path":   outputPath,
		"resolved_path": resolvedInputPath,
	}, nil
}

func (s *Server) handleExtractTextFromDocx(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Extracting text from Word document", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	// Optional output path for saving text to file
	outputPath, _ := params["output_path"].(string)

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}

	// Extract text
	text, err := s.wordProcessor.ExtractText(resolvedInputPath)
	if err != nil {
		return nil, err
	}

	// Save to file if output path provided
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(text), 0644); err != nil {
			return nil, fmt.Errorf("failed to write text file: %w", err)
		}
	}

	return map[string]interface{}{
		"status":        "success",
		"text":          text,
		"text_length":   len(text),
		"output_path":   outputPath,
		"resolved_path": resolvedInputPath,
	}, nil
}

func (s *Server) handleGetDocxMetadata(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Getting Word document metadata", zap.Any("params", params))

	// Extract parameters (supports both file paths and DEVONthink links)
	inputPath, ok := params["input_path"].(string)
	if !ok || inputPath == "" {
		return nil, fmt.Errorf("input_path is required")
	}

	// Resolve DEVONthink link to file path
	resolvedInputPath, resolveErr := s.dtResolver.ResolveLinkToPath(inputPath)
	if resolveErr != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", resolveErr)
	}

	// Get metadata
	metadata, err := s.wordProcessor.GetMetadata(resolvedInputPath)
	if err != nil {
		return nil, err
	}

	metadata["status"] = "success"
	metadata["resolved_path"] = resolvedInputPath

	return metadata, nil
}

func (s *Server) handleIngestDocument(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Ingesting document", zap.Any("params", params))

	// Extract parameters
	filePath, ok := params["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Create process request
	req := &pipeline.ProcessRequest{
		FilePath: filePath,
		DocType:  extractParamString(params, "document_type", ""),
		Metadata: extractParamMap(params, "metadata", make(map[string]interface{})),
		Options: pipeline.ProcessingOptions{
			OCR:           extractParamBool(params, "ocr", true),
			ExtractTables: extractParamBool(params, "extract_tables", false),
			Vectorize:     extractParamBool(params, "vectorize", false),
			StoreOriginal: extractParamBool(params, "store_original", true),
		},
	}

	// Process document
	resp, err := s.processingOrchestrator.ProcessDocument(ctx, req)
	if err != nil {
		s.logger.Error("Document processing failed", zap.Error(err))
		return nil, fmt.Errorf("document processing failed: %w", err)
	}

	return resp, nil
}

func (s *Server) handleIngestBatch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	s.logger.Info("Ingesting batch of documents", zap.Any("params", params))

	// Extract parameters
	filePaths, ok := params["file_paths"].([]interface{})
	if !ok || len(filePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required and must be a non-empty list")
	}

	// Convert []interface{} to []string
	paths := make([]string, len(filePaths))
	for i, p := range filePaths {
		path, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("all file_paths must be strings, index %d is not", i)
		}
		paths[i] = path
	}

	// Extract optional parameters
	onError := extractParamString(params, "on_error", "continue") // continue or stop
	ocr := extractParamBool(params, "ocr", true)
	extractTables := extractParamBool(params, "extract_tables", false)
	vectorize := extractParamBool(params, "vectorize", false)
	storeOriginal := extractParamBool(params, "store_original", true)

	// Create batch request
	type BatchResult struct {
		FilePath string      `json:"file_path"`
		Status   string      `json:"status"`
		DocID    string      `json:"doc_id,omitempty"`
		Error    string      `json:"error,omitempty"`
		Duration time.Duration `json:"duration,omitempty"`
	}

	results := make([]BatchResult, 0, len(paths))
	successCount := 0
	failureCount := 0

	// Process each file sequentially
	for i, filePath := range paths {
		batchStart := time.Now()

		// Check if file exists
		if _, err := os.Stat(filePath); err != nil {
			s.logger.Warn("File not found in batch", zap.String("path", filePath), zap.Int("index", i))
			results = append(results, BatchResult{
				FilePath: filePath,
				Status:   "failed",
				Error:    fmt.Sprintf("file not found: %v", err),
				Duration: time.Since(batchStart),
			})
			failureCount++

			if onError == "stop" {
				return map[string]interface{}{
					"status":       "stopped",
					"success_count": successCount,
					"failure_count": failureCount,
					"results":      results,
					"stopped_at":    i,
					"message":       "Batch stopped due to error (on_error=stop)",
				}, nil
			}
			continue
		}

		// Process document
		req := &pipeline.ProcessRequest{
			FilePath: filePath,
			Options: pipeline.ProcessingOptions{
				OCR:           ocr,
				ExtractTables: extractTables,
				Vectorize:     vectorize,
				StoreOriginal: storeOriginal,
			},
		}

		resp, err := s.processingOrchestrator.ProcessDocument(ctx, req)
		if err != nil {
			s.logger.Error("Document processing failed in batch",
				zap.String("path", filePath),
				zap.Int("index", i),
				zap.Error(err))
			results = append(results, BatchResult{
				FilePath: filePath,
				Status:   "failed",
				Error:    err.Error(),
				Duration: time.Since(batchStart),
			})
			failureCount++

			if onError == "stop" {
				return map[string]interface{}{
					"status":        "stopped",
					"success_count": successCount,
					"failure_count": failureCount,
					"results":       results,
					"stopped_at":    i,
					"message":       "Batch stopped due to error (on_error=stop)",
				}, nil
			}
			continue
		}

		s.logger.Info("Document processed in batch",
			zap.String("path", filePath),
			zap.String("doc_id", resp.DocumentID),
			zap.Duration("duration", time.Since(batchStart)))

		results = append(results, BatchResult{
			FilePath: filePath,
			Status:   resp.Status,
			DocID:    resp.DocumentID,
			Duration: time.Since(batchStart),
		})
		successCount++
	}

	return map[string]interface{}{
		"status":        "completed",
		"total_count":   len(paths),
		"success_count": successCount,
		"failure_count": failureCount,
		"results":       results,
	}, nil
}

// Helper functions for parameter extraction
func extractParamString(params map[string]interface{}, key string, defaultValue string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return defaultValue
}

func extractParamBool(params map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return defaultValue
}

func extractParamMap(params map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	if v, ok := params[key].(map[string]interface{}); ok {
		return v
	}
	return defaultValue
}
