package pdfa

import "image"

// ColorSpaceType defines the color space for PDF/A output
type ColorSpaceType string

const (
	ColorSpaceDeviceRGB    ColorSpaceType = "DeviceRGB"
	ColorSpaceDeviceCMYK   ColorSpaceType = "DeviceCMYK"
	ColorSpaceICCBasedRGB  ColorSpaceType = "ICCBasedRGB"
	ColorSpaceICCBasedCMYK ColorSpaceType = "ICCBasedCMYK"
)

// DownsamplingMethod defines how to downsample images
type DownsamplingMethod string

const (
	DownsamplingMethodBicubic    DownsamplingMethod = "bicubic"    // High quality (CatmullRom)
	DownsamplingMethodAverage    DownsamplingMethod = "average"    // Balanced (ApproxBiLinear)
	DownsamplingMethodSubsample  DownsamplingMethod = "subsample"  // Fast (NearestNeighbor)
)

// QualityProfile defines preset quality settings
type QualityProfile string

const (
	QualityProfileDefault     QualityProfile = "default"      // Balanced (150 DPI, 90% quality)
	QualityProfileHighQuality QualityProfile = "high-quality" // Archival (300 DPI, 95% quality)
	QualityProfileCompressed  QualityProfile = "compressed"   // Distribution (100 DPI, 80% quality)
)

// PDFA2bConfig holds configuration for PDF/A-2b compliant output
type PDFA2bConfig struct {
	// Color space configuration
	ColorSpace ColorSpaceType

	// Image processing
	DownsamplingMethod  DownsamplingMethod
	DownsampleThreshold int    // DPI threshold - only downsample if source exceeds this
	TargetDPI           int    // Target DPI after downsampling (72-300)
	ImageQuality        int    // JPEG quality 1-100 (typically 80-95 for documents)

	// PDF optimization
	Linearize   bool // Optimize for streaming/web viewing
	CompressAll bool // Compress all content (streams, text)

	// Metadata
	Title       string
	Author      string
	Subject     string
	Keywords    string
	Creator     string
	Producer    string

	// Compliance
	ValidateAfterCreation bool // Run compliance check after creation
	StrictMode            bool // Strict PDF/A-2b compliance (no warnings)

	// Processing
	RemoveDuplicates bool // Remove duplicate images
	RemoveAlpha      bool // Remove transparency (required for PDF/A)
	ConvertToRGB     bool // Convert CMYK to RGB if needed
}

// NewDefaultPDFA2bConfig creates a balanced configuration for general use
func NewDefaultPDFA2bConfig() *PDFA2bConfig {
	return &PDFA2bConfig{
		ColorSpace:          ColorSpaceDeviceRGB,
		DownsamplingMethod:  DownsamplingMethodAverage,
		DownsampleThreshold: 150, // Only downsample from 150+ DPI
		TargetDPI:           150,
		ImageQuality:        90,
		Linearize:           false,
		CompressAll:         true,
		RemoveDuplicates:    false,
		RemoveAlpha:         true,
		ConvertToRGB:        false,
		ValidateAfterCreation: true,
		StrictMode:          false,
	}
}

// NewHighQualityPDFA2bConfig creates a configuration for archival/legal use
func NewHighQualityPDFA2bConfig() *PDFA2bConfig {
	return &PDFA2bConfig{
		ColorSpace:          ColorSpaceICCBasedRGB,
		DownsamplingMethod:  DownsamplingMethodBicubic,
		DownsampleThreshold: 300, // Preserve high quality
		TargetDPI:           300,
		ImageQuality:        95,
		Linearize:           true,
		CompressAll:         true,
		RemoveDuplicates:    true,
		RemoveAlpha:         true,
		ConvertToRGB:        true,
		ValidateAfterCreation: true,
		StrictMode:          true,
	}
}

// NewCompressedPDFA2bConfig creates a configuration for distribution/web use
func NewCompressedPDFA2bConfig() *PDFA2bConfig {
	return &PDFA2bConfig{
		ColorSpace:          ColorSpaceDeviceRGB,
		DownsamplingMethod:  DownsamplingMethodSubsample,
		DownsampleThreshold: 100,
		TargetDPI:           100,
		ImageQuality:        80,
		Linearize:           true,
		CompressAll:         true,
		RemoveDuplicates:    true,
		RemoveAlpha:         true,
		ConvertToRGB:        false,
		ValidateAfterCreation: true,
		StrictMode:          false,
	}
}

// ImageMetadata holds extracted information about an image
type ImageMetadata struct {
	FilePath      string
	Width         int
	Height        int
	ColorSpace    string // "RGB", "RGBA", "CMYK", "Gray"
	HasAlpha      bool
	EstimatedDPI  int // Estimated from physical size if available
	FileSize      int64
	Format        string // "JPEG", "PNG", "TIFF", "GIF", "BMP"
	Bounds        image.Rectangle
	NeedsDownsampling bool // Whether this image should be downsampled
	NewWidth      int    // Target width after downsampling
	NewHeight     int    // Target height after downsampling
}

// ComplianceReport holds results of PDF/A-2b compliance check
type ComplianceReport struct {
	IsCompliant           bool
	Version               string // "PDF/A-2b"
	HasTransparency       bool
	HasEmbeddedFonts      bool
	HasNonEmbeddedFonts   bool
	HasJavaScript         bool
	HasEncryption         bool
	ColorSpace            string
	ImageCount            int
	IssuesFound           []string // Compliance issues
	WarningsFound         []string // Non-blocking warnings
}
