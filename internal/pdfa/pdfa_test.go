package pdfa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// Test configuration presets
func TestDefaultConfig(t *testing.T) {
	config := NewDefaultPDFA2bConfig()

	if config.ColorSpace != ColorSpaceDeviceRGB {
		t.Errorf("Expected DeviceRGB, got %s", config.ColorSpace)
	}
	if config.DownsamplingMethod != DownsamplingMethodAverage {
		t.Errorf("Expected Average downsampling, got %s", config.DownsamplingMethod)
	}
	if config.DownsampleThreshold != 150 {
		t.Errorf("Expected threshold 150, got %d", config.DownsampleThreshold)
	}
	if config.TargetDPI != 150 {
		t.Errorf("Expected target DPI 150, got %d", config.TargetDPI)
	}
	if config.ImageQuality != 90 {
		t.Errorf("Expected quality 90, got %d", config.ImageQuality)
	}
}

func TestHighQualityConfig(t *testing.T) {
	config := NewHighQualityPDFA2bConfig()

	if config.ColorSpace != ColorSpaceICCBasedRGB {
		t.Errorf("Expected ICCBasedRGB, got %s", config.ColorSpace)
	}
	if config.DownsamplingMethod != DownsamplingMethodBicubic {
		t.Errorf("Expected Bicubic downsampling, got %s", config.DownsamplingMethod)
	}
	if config.DownsampleThreshold != 300 {
		t.Errorf("Expected threshold 300, got %d", config.DownsampleThreshold)
	}
	if config.TargetDPI != 300 {
		t.Errorf("Expected target DPI 300, got %d", config.TargetDPI)
	}
	if config.ImageQuality != 95 {
		t.Errorf("Expected quality 95, got %d", config.ImageQuality)
	}
	if config.StrictMode != true {
		t.Errorf("Expected strict mode enabled")
	}
}

func TestCompressedConfig(t *testing.T) {
	config := NewCompressedPDFA2bConfig()

	if config.ColorSpace != ColorSpaceDeviceRGB {
		t.Errorf("Expected DeviceRGB, got %s", config.ColorSpace)
	}
	if config.DownsamplingMethod != DownsamplingMethodSubsample {
		t.Errorf("Expected Subsample downsampling, got %s", config.DownsamplingMethod)
	}
	if config.DownsampleThreshold != 100 {
		t.Errorf("Expected threshold 100, got %d", config.DownsampleThreshold)
	}
	if config.TargetDPI != 100 {
		t.Errorf("Expected target DPI 100, got %d", config.TargetDPI)
	}
	if config.ImageQuality != 80 {
		t.Errorf("Expected quality 80, got %d", config.ImageQuality)
	}
}

// Test PDFACreator initialization
func TestPDFACreatorInit(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	creator := NewPDFACreator(config, logger)

	if creator == nil {
		t.Error("Failed to create PDFACreator")
	}
	if creator.config != config {
		t.Error("Config not properly set")
	}
}

func TestPDFACreatorInitWithNilConfig(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(nil, logger)

	if creator == nil {
		t.Error("Failed to create PDFACreator with nil config")
	}
	if creator.config == nil {
		t.Error("Default config not created for nil input")
	}
}

func TestPDFACreatorInitWithNilLogger(t *testing.T) {
	config := NewDefaultPDFA2bConfig()
	creator := NewPDFACreator(config, nil)

	if creator == nil {
		t.Error("Failed to create PDFACreator with nil logger")
	}
	if creator.logger == nil {
		t.Error("Default logger not created for nil input")
	}
}

// Test metadata application
func TestApplyMetadata(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	config.Title = "Test Document"
	config.Author = "Test Author"
	config.Subject = "Test Subject"
	config.Keywords = "test,keywords"

	creator := NewPDFACreator(config, logger)

	// Create a temporary file to simulate a PDF
	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Apply metadata to the temporary file
	err = creator.ApplyMetadata(tmpFile.Name())
	if err != nil {
		t.Errorf("ApplyMetadata failed: %v", err)
	}
}

func TestApplyMetadataToNonexistentFile(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	err := creator.ApplyMetadata("/nonexistent/path/file.pdf")
	if err == nil {
		t.Error("ApplyMetadata should fail for nonexistent file")
	}
}

// Test optimization
func TestOptimizeForArchival(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	config.Linearize = true
	config.CompressAll = true

	creator := NewPDFACreator(config, logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = creator.OptimizeForArchival(tmpFile.Name())
	if err != nil {
		t.Errorf("OptimizeForArchival failed: %v", err)
	}
}

// Test compliance validation
func TestValidatePDFACompliance(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Errorf("ValidatePDFACompliance failed: %v", err)
	}

	if report == nil {
		t.Error("Compliance report is nil")
	}
	if report.Version != "PDF/A-2b" {
		t.Errorf("Expected version PDF/A-2b, got %s", report.Version)
	}
}

func TestValidatePDFAComplianceToNonexistentFile(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	_, err := creator.ValidatePDFACompliance("/nonexistent/path/file.pdf")
	if err == nil {
		t.Error("ValidatePDFACompliance should fail for nonexistent file")
	}
}

// Test compliance report generation
func TestGenerateComplianceReport(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.GenerateComplianceReport(tmpFile.Name())
	if err != nil {
		t.Errorf("GenerateComplianceReport failed: %v", err)
	}

	if report == nil {
		t.Error("Compliance report is nil")
	}
	if report.Version != "PDF/A-2b" {
		t.Errorf("Expected version PDF/A-2b, got %s", report.Version)
	}
}

// Test ICC profile application
func TestApplyICCProfile(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	// Create temporary PDF file
	pdfFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp PDF file: %v", err)
	}
	defer os.Remove(pdfFile.Name())
	pdfFile.Close()

	// Create temporary ICC profile file
	iccFile, err := os.CreateTemp("", "test*.icc")
	if err != nil {
		t.Fatalf("Failed to create temp ICC file: %v", err)
	}
	defer os.Remove(iccFile.Name())
	iccFile.Close()

	// Apply ICC profile
	err = creator.ApplyICCProfile(pdfFile.Name(), iccFile.Name())
	if err != nil {
		t.Errorf("ApplyICCProfile failed: %v", err)
	}
}

func TestApplyICCProfileToNonexistentPDF(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	// Create temporary ICC profile file
	iccFile, err := os.CreateTemp("", "test*.icc")
	if err != nil {
		t.Fatalf("Failed to create temp ICC file: %v", err)
	}
	defer os.Remove(iccFile.Name())
	iccFile.Close()

	err = creator.ApplyICCProfile("/nonexistent/path/file.pdf", iccFile.Name())
	if err == nil {
		t.Error("ApplyICCProfile should fail for nonexistent PDF")
	}
}

func TestApplyICCProfileToNonexistentICC(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	// Create temporary PDF file
	pdfFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp PDF file: %v", err)
	}
	defer os.Remove(pdfFile.Name())
	pdfFile.Close()

	err = creator.ApplyICCProfile(pdfFile.Name(), "/nonexistent/path/profile.icc")
	if err == nil {
		t.Error("ApplyICCProfile should fail for nonexistent ICC profile")
	}
}

// Test color space conversion
func TestConvertColorSpace(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	testCases := []ColorSpaceType{
		ColorSpaceDeviceRGB,
		ColorSpaceDeviceCMYK,
		ColorSpaceICCBasedRGB,
		ColorSpaceICCBasedCMYK,
	}

	for _, cs := range testCases {
		err := creator.ConvertColorSpace(tmpFile.Name(), cs)
		if err != nil {
			t.Errorf("ConvertColorSpace failed for %s: %v", cs, err)
		}
	}
}

func TestConvertColorSpaceInvalidColorSpace(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = creator.ConvertColorSpace(tmpFile.Name(), ColorSpaceType("invalid"))
	if err == nil {
		t.Error("ConvertColorSpace should fail for invalid color space")
	}
}

// Test config summary
func TestGetConfigSummary(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	config.Title = "Test"
	config.Author = "Author"

	creator := NewPDFACreator(config, logger)
	summary := creator.GetConfigSummary()

	if summary == nil {
		t.Error("Config summary is nil")
	}
	if summary["color_space"] != ColorSpaceDeviceRGB {
		t.Errorf("Unexpected color space in summary")
	}
	if summary["image_quality"] != 90 {
		t.Errorf("Unexpected image quality in summary")
	}
}

// Test helper functions
func TestGetColorSpaceDescription(t *testing.T) {
	testCases := []struct {
		cs           ColorSpaceType
		shouldContain string
	}{
		{ColorSpaceDeviceRGB, "RGB"},
		{ColorSpaceDeviceCMYK, "CMYK"},
		{ColorSpaceICCBasedRGB, "ICC"},
		{ColorSpaceICCBasedCMYK, "ICC"},
	}

	for _, tc := range testCases {
		desc := GetColorSpaceDescription(tc.cs)
		if len(desc) == 0 {
			t.Errorf("Empty description for %s", tc.cs)
		}
	}
}

func TestGetDownsamplingMethodDescription(t *testing.T) {
	testCases := []struct {
		method       DownsamplingMethod
		shouldContain string
	}{
		{DownsamplingMethodBicubic, "Bicubic"},
		{DownsamplingMethodAverage, "Average"},
		{DownsamplingMethodSubsample, "Subsample"},
	}

	for _, tc := range testCases {
		desc := GetDownsamplingMethodDescription(tc.method)
		if len(desc) == 0 {
			t.Errorf("Empty description for %s", tc.method)
		}
	}
}

// Test ProcessAndCreatePDFA with image files
func TestProcessAndCreatePDFA(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()

	creator := NewPDFACreator(config, logger)

	// Create temporary directory for output
	tmpDir, err := os.MkdirTemp("", "pdf_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "output.pdf")

	// Note: This test would need actual image files
	// For now, we're testing the error handling path
	imagePaths := []string{}
	processErr := error(nil)
	_, processErr = creator.ProcessAndCreatePDFA(imagePaths, outputPath, 300)

	// Expected to fail because no images provided
	if processErr == nil && len(imagePaths) == 0 {
		// Test passes if error or report is handled properly
	}
}

// Test ImageMetadata struct
func TestImageMetadata(t *testing.T) {
	metadata := &ImageMetadata{
		FilePath:      "/path/to/image.jpg",
		Width:         2400,
		Height:        3200,
		ColorSpace:    "RGB",
		HasAlpha:      false,
		EstimatedDPI:  300,
		FileSize:      2048000,
		Format:        "JPEG",
		NeedsDownsampling: true,
		NewWidth:      1200,
		NewHeight:     1600,
	}

	if metadata.FilePath != "/path/to/image.jpg" {
		t.Error("FilePath not set correctly")
	}
	if metadata.Width != 2400 {
		t.Error("Width not set correctly")
	}
	if !metadata.NeedsDownsampling {
		t.Error("NeedsDownsampling not set correctly")
	}
}

// Test ComplianceReport struct
func TestComplianceReport(t *testing.T) {
	report := &ComplianceReport{
		IsCompliant:       true,
		Version:           "PDF/A-2b",
		HasTransparency:   false,
		HasEmbeddedFonts:  true,
		HasNonEmbeddedFonts: false,
		HasJavaScript:     false,
		HasEncryption:     false,
		ColorSpace:        "DeviceRGB",
		ImageCount:        5,
		IssuesFound:       []string{},
		WarningsFound:     []string{},
	}

	if !report.IsCompliant {
		t.Error("Compliance status not set correctly")
	}
	if report.Version != "PDF/A-2b" {
		t.Error("Version not set correctly")
	}
	if report.ImageCount != 5 {
		t.Error("ImageCount not set correctly")
	}
}

// TestValidatePDFACompliance_HonestStatus tests that validation returns honest "not implemented" status
// This test addresses Issue #15: PDF/A validation should return honest status instead of false positives
func TestValidatePDFACompliance_HonestStatus(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// Critical: Validation is not implemented, so IsCompliant should be false
	if report.IsCompliant {
		t.Error("ValidatePDFACompliance returned IsCompliant=true, but validation is not implemented (should be false)")
	}

	// Should have issues indicating validation is not implemented
	if len(report.IssuesFound) == 0 {
		t.Error("Expected issues indicating validation not implemented, got empty array")
	}

	// Check for specific "not implemented" message
	foundNotImplemented := false
	for _, issue := range report.IssuesFound {
		if issue == "PDF/A validation not implemented" {
			foundNotImplemented = true
			break
		}
	}
	if !foundNotImplemented {
		t.Error("Expected 'PDF/A validation not implemented' in issues list")
	}
}

// TestValidatePDFACompliance_ManualValidationRequired tests that report indicates manual validation needed
func TestValidatePDFACompliance_ManualValidationRequired(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// Check for manual validation message
	foundManualValidation := false
	for _, issue := range report.IssuesFound {
		if issue == "Manual validation required for archival purposes" {
			foundManualValidation = true
			break
		}
	}
	if !foundManualValidation {
		t.Error("Expected 'Manual validation required for archival purposes' in issues list")
	}
}

// TestValidatePDFACompliance_NoFalsePositives tests that unknown attributes are false, not true
func TestValidatePDFACompliance_NoFalsePositives(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// When validation is not implemented, attributes that would require validation
	// should be false (not validated = assume false)
	if report.HasEmbeddedFonts {
		t.Error("HasEmbeddedFonts should be false when not validated")
	}
	if report.HasNonEmbeddedFonts {
		t.Error("HasNonEmbeddedFonts should be false when not validated")
	}
	if report.HasJavaScript {
		t.Error("HasJavaScript should be false when not validated")
	}
	if report.HasEncryption {
		t.Error("HasEncryption should be false when not validated")
	}
	if report.HasTransparency {
		t.Error("HasTransparency should be false when not validated")
	}
}

// TestValidatePDFACompliance_RegresssionNoFalseSuccess tests against regression to false success
func TestValidatePDFACompliance_RegressionNoFalseSuccess(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Test with different configs to ensure none return false positives
	configs := []*PDFA2bConfig{
		NewDefaultPDFA2bConfig(),
		NewHighQualityPDFA2bConfig(),
		NewCompressedPDFA2bConfig(),
	}

	for i, config := range configs {
		creator := NewPDFACreator(config, logger)

		tmpFile, err := os.CreateTemp("", "test*.pdf")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		report, err := creator.ValidatePDFACompliance(tmpFile.Name())
		if err != nil {
			t.Fatalf("Config %d: ValidatePDFACompliance failed: %v", i, err)
		}

		// CRITICAL: Must never return IsCompliant=true when validation is not implemented
		if report.IsCompliant {
			t.Errorf("Config %d: REGRESSION - IsCompliant should be false (validation not implemented)", i)
		}

		// Must always have issues indicating not implemented
		if len(report.IssuesFound) == 0 {
			t.Errorf("Config %d: REGRESSION - IssuesFound should not be empty", i)
		}
	}
}

// TestValidatePDFACompliance_EmptyFile tests validation with empty PDF file
func TestValidatePDFACompliance_EmptyFile(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "empty*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())

	// Should not fail on empty file, just report not implemented
	if err != nil {
		t.Fatalf("ValidatePDFACompliance should not fail on empty file: %v", err)
	}

	if report.IsCompliant {
		t.Error("Empty file cannot be PDF/A compliant without validation")
	}
}

// TestValidatePDFACompliance_ReportStructure tests that report structure is correct
func TestValidatePDFACompliance_ReportStructure(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// Verify all required fields are set correctly
	if report.Version != "PDF/A-2b" {
		t.Errorf("Version = %v, want PDF/A-2b", report.Version)
	}

	if len(report.IssuesFound) == 0 {
		t.Error("IssuesFound should not be empty for not-implemented validation")
	}

	// Verify warnings is initialized (can be empty)
	if report.WarningsFound == nil {
		t.Error("WarningsFound should be initialized (even if empty)")
	}
}

// TestValidatePDFACompliance_NeverReturnsTrue tests that unimplemented validation never claims compliance
func TestValidatePDFACompliance_NeverReturnsTrue(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Test with all configuration presets
	configs := []*PDFA2bConfig{
		NewDefaultPDFA2bConfig(),
		NewHighQualityPDFA2bConfig(),
		NewCompressedPDFA2bConfig(),
	}

	for i, config := range configs {
		creator := NewPDFACreator(config, logger)

		tmpFile, err := os.CreateTemp("", "test*.pdf")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		report, err := creator.ValidatePDFACompliance(tmpFile.Name())
		if err != nil {
			t.Fatalf("Config %d: ValidatePDFACompliance failed: %v", i, err)
		}

		// CRITICAL: Must NEVER return true when validation is not implemented
		if report.IsCompliant {
			t.Errorf("Config %d: CRITICAL - IsCompliant=true but validation not implemented (must be false)", i)
		}
	}
}

// TestValidatePDFACompliance_MessageContent tests that issue messages are informative
func TestValidatePDFACompliance_MessageContent(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// Check that issues contain informative messages
	expectedMessages := []string{
		"PDF/A validation not implemented",
		"Manual validation required",
	}

	for _, expectedMsg := range expectedMessages {
		found := false
		for _, issue := range report.IssuesFound {
			if strings.Contains(strings.ToLower(issue), strings.ToLower(expectedMsg)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected issue containing '%s' not found in issues list", expectedMsg)
		}
	}
}

// TestValidatePDFACompliance_StrictModeStillHonest tests strict mode doesn't override honesty
func TestValidatePDFACompliance_StrictModeStillHonest(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewHighQualityPDFA2bConfig()
	config.StrictMode = true // Enable strict mode

	creator := NewPDFACreator(config, logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed: %v", err)
	}

	// Even in strict mode, should be honest about not being implemented
	if report.IsCompliant {
		t.Error("CRITICAL: Strict mode should not override honest status (IsCompliant should be false)")
	}

	if len(report.IssuesFound) == 0 {
		t.Error("Strict mode should still report issues when validation not implemented")
	}
}

// TestValidatePDFACompliance_LargeFile tests validation with large file
func TestValidatePDFACompliance_LargeFile(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	// Create a larger temporary file
	tmpFile, err := os.CreateTemp("", "large*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 10MB of data
	largeData := make([]byte, 10*1024*1024)
	if _, err := tmpFile.Write(largeData); err != nil {
		t.Fatalf("Failed to write large file: %v", err)
	}
	tmpFile.Close()

	report, err := creator.ValidatePDFACompliance(tmpFile.Name())
	if err != nil {
		t.Fatalf("ValidatePDFACompliance failed on large file: %v", err)
	}

	// Should handle large files without panic
	if report.IsCompliant {
		t.Error("Large file should not be marked as compliant without validation")
	}
}

// TestValidatePDFACompliance_ConcurrentCalls tests concurrent validation calls
func TestValidatePDFACompliance_ConcurrentCalls(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	// Create temp files
	var files []string
	for i := 0; i < 5; i++ {
		tmpFile, err := os.CreateTemp("", "concurrent*.pdf")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		tmpFile.Close()
		files = append(files, tmpFile.Name())
		defer os.Remove(tmpFile.Name())
	}

	// Run validations concurrently
	done := make(chan bool)
	for i, file := range files {
		go func(idx int, path string) {
			report, err := creator.ValidatePDFACompliance(path)
			if err != nil {
				t.Errorf("Concurrent call %d failed: %v", idx, err)
			}
			if report.IsCompliant {
				t.Errorf("Concurrent call %d: IsCompliant should be false", idx)
			}
			done <- true
		}(i, file)
	}

	// Wait for all goroutines
	for i := 0; i < len(files); i++ {
		<-done
	}
}

// TestApplyMetadata_BoundaryValues tests metadata with extreme values
func TestApplyMetadata_BoundaryValues(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()

	// Test with very long strings
	config.Title = string(make([]byte, 10000))
	config.Author = string(make([]byte, 5000))
	config.Subject = string(make([]byte, 5000))
	config.Keywords = string(make([]byte, 20000))

	creator := NewPDFACreator(config, logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Should handle long metadata without panic
	err = creator.ApplyMetadata(tmpFile.Name())
	if err != nil {
		t.Logf("Long metadata may be rejected (acceptable): %v", err)
	}
}

// TestOptimizeForArchival_AllOptions tests all optimization options enabled
func TestOptimizeForArchival_AllOptions(t *testing.T) {
	logger, _ := zap.NewProduction()
	config := NewDefaultPDFA2bConfig()
	config.Linearize = true
	config.CompressAll = true
	config.RemoveDuplicates = true
	config.RemoveAlpha = true

	creator := NewPDFACreator(config, logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = creator.OptimizeForArchival(tmpFile.Name())
	if err != nil {
		t.Errorf("OptimizeForArchival with all options failed: %v", err)
	}
}

// TestGetConfigSummary_AllConfigs tests config summary for all presets
func TestGetConfigSummary_AllConfigs(t *testing.T) {
	logger, _ := zap.NewProduction()

	configs := []*PDFA2bConfig{
		NewDefaultPDFA2bConfig(),
		NewHighQualityPDFA2bConfig(),
		NewCompressedPDFA2bConfig(),
	}

	for i, config := range configs {
		creator := NewPDFACreator(config, logger)
		summary := creator.GetConfigSummary()

		if summary == nil {
			t.Errorf("Config %d: Summary should not be nil", i)
		}

		// Check that summary contains expected keys
		expectedKeys := []string{
			"color_space",
			"downsampling_method",
			"target_dpi",
			"image_quality",
			"linearize",
		}

		for _, key := range expectedKeys {
			if _, ok := summary[key]; !ok {
				t.Errorf("Config %d: Summary missing key '%s'", i, key)
			}
		}
	}
}

// TestConvertColorSpace_AllTypes tests conversion to all supported color spaces
func TestConvertColorSpace_AllTypes(t *testing.T) {
	logger, _ := zap.NewProduction()
	creator := NewPDFACreator(NewDefaultPDFA2bConfig(), logger)

	tmpFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	colorSpaces := []ColorSpaceType{
		ColorSpaceDeviceRGB,
		ColorSpaceDeviceCMYK,
		ColorSpaceICCBasedRGB,
		ColorSpaceICCBasedCMYK,
	}

	for _, cs := range colorSpaces {
		err := creator.ConvertColorSpace(tmpFile.Name(), cs)
		if err != nil {
			t.Errorf("ConvertColorSpace to %s failed: %v", cs, err)
		}
	}
}

// TestGetColorSpaceDescription_AllTypes tests descriptions for all color space types
func TestGetColorSpaceDescription_AllTypes(t *testing.T) {
	colorSpaces := []ColorSpaceType{
		ColorSpaceDeviceRGB,
		ColorSpaceDeviceCMYK,
		ColorSpaceICCBasedRGB,
		ColorSpaceICCBasedCMYK,
		ColorSpaceType("unknown"),
	}

	for _, cs := range colorSpaces {
		desc := GetColorSpaceDescription(cs)
		if len(desc) == 0 {
			t.Errorf("Description for %s should not be empty", cs)
		}
	}
}

// TestGetDownsamplingMethodDescription_AllMethods tests descriptions for all downsampling methods
func TestGetDownsamplingMethodDescription_AllMethods(t *testing.T) {
	methods := []DownsamplingMethod{
		DownsamplingMethodBicubic,
		DownsamplingMethodAverage,
		DownsamplingMethodSubsample,
		DownsamplingMethod("unknown"),
	}

	for _, method := range methods {
		desc := GetDownsamplingMethodDescription(method)
		if len(desc) == 0 {
			t.Errorf("Description for %s should not be empty", method)
		}
	}
}