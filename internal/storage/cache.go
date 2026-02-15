package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Cache manages file storage and caching
type Cache struct {
	tempDir string
	files   map[string]*CachedFile
	mu      sync.RWMutex
}

// CachedFile represents a cached file with metadata
type CachedFile struct {
	ID        string
	Path      string
	Size      int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewCache creates a new cache instance
func NewCache(tempDir string) *Cache {
	// Create temp directory if it doesn't exist
	_ = os.MkdirAll(tempDir, 0755)

	return &Cache{
		tempDir: tempDir,
		files:   make(map[string]*CachedFile),
	}
}

// SaveFile saves a file to cache and returns the path
func (c *Cache) SaveFile(id string, data []byte) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := filepath.Join(c.tempDir, id)

	// Write file to disk
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Track cached file
	c.files[id] = &CachedFile{
		ID:        id,
		Path:      filePath,
		Size:      int64(len(data)),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour expiration
	}

	return filePath, nil
}

// GetFile retrieves a cached file
func (c *Cache) GetFile(id string) (*CachedFile, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.files[id]
	if !exists {
		return nil, fmt.Errorf("file not found in cache: %s", id)
	}

	// Check if file still exists on disk
	if _, err := os.Stat(cached.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("cached file no longer exists: %s", id)
	}

	return cached, nil
}

// ReadFile reads a cached file's contents
func (c *Cache) ReadFile(id string) ([]byte, error) {
	cached, err := c.GetFile(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cached.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached file: %w", err)
	}

	return data, nil
}

// CopyFile copies a file from source to cache
func (c *Cache) CopyFile(id string, sourcePath string) (string, error) {
	src, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	destPath := filepath.Join(c.tempDir, id)
	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Get file size for tracking
	stat, err := os.Stat(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.files[id] = &CachedFile{
		ID:        id,
		Path:      destPath,
		Size:      stat.Size(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return destPath, nil
}

// DeleteFile removes a file from cache
func (c *Cache) DeleteFile(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, exists := c.files[id]
	if !exists {
		return fmt.Errorf("file not found in cache: %s", id)
	}

	if err := os.Remove(cached.Path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	delete(c.files, id)
	return nil
}

// ListFiles returns all cached files
func (c *Cache) ListFiles() []*CachedFile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	files := make([]*CachedFile, 0, len(c.files))
	for _, f := range c.files {
		files = append(files, f)
	}
	return files
}

// CleanupExpired removes expired files from cache
func (c *Cache) CleanupExpired() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for id, cached := range c.files {
		if now.After(cached.ExpiresAt) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		cached := c.files[id]
		_ = os.Remove(cached.Path)
		delete(c.files, id)
	}

	return nil
}

// ClearAll removes all cached files
func (c *Cache) ClearAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, cached := range c.files {
		_ = os.Remove(cached.Path)
		delete(c.files, id)
	}

	return nil
}

// =====================================================
// R2 Uploader - Cloudflare R2 Integration
// =====================================================

// R2Uploader handles uploads to Cloudflare R2 buckets
type R2Uploader struct {
	s3Client *s3.Client
	bucket   string
	logger   *zap.Logger
}

// NewR2Uploader creates a new R2 uploader
func NewR2Uploader(accessKeyID, secretAccessKey, endpoint, bucket string, logger *zap.Logger) (*R2Uploader, error) {
	if accessKeyID == "" || secretAccessKey == "" || endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("missing required R2 configuration")
	}

	// Create credentials
	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")

	// Create custom resolver for R2 endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: "auto",
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown service")
	})

	// Create config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	logger.Info("R2 uploader initialized",
		zap.String("bucket", bucket),
		zap.String("endpoint", endpoint))

	return &R2Uploader{
		s3Client: s3Client,
		bucket:   bucket,
		logger:   logger,
	}, nil
}

// UploadOriginal uploads a document's original file to R2
// Path: /originals/{docID}/{filename}
func (u *R2Uploader) UploadOriginal(ctx context.Context, filePath string, docID string) (string, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		u.logger.Error("Failed to read file", zap.String("path", filePath), zap.Error(err))
		return "", fmt.Errorf("read file: %w", err)
	}

	filename := filepath.Base(filePath)
	key := fmt.Sprintf("originals/%s/%s", docID, filename)

	// Upload to R2
	_, err = u.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
		Metadata: map[string]string{
			"original-name": filename,
			"doc-id":        docID,
			"upload-time":   time.Now().Format(time.RFC3339),
		},
	})

	if err != nil {
		u.logger.Error("Failed to upload to R2", zap.String("key", key), zap.Error(err))
		return "", fmt.Errorf("upload to R2: %w", err)
	}

	u.logger.Info("Original document uploaded",
		zap.String("key", key),
		zap.String("doc_id", docID),
		zap.Int64("size", int64(len(data))))

	return key, nil
}

// UploadProcessed uploads processed output to R2
// Path: /processed/{docID}/{outputType}
func (u *R2Uploader) UploadProcessed(ctx context.Context, data []byte, docID string, outputType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("no data to upload")
	}

	// Determine file extension based on output type
	ext := ""
	switch outputType {
	case "docling_full", "mistral_ocr", "metadata":
		ext = ".json"
	case "docling_markdown":
		ext = ".md"
	case "docling_structure", "docling_tables":
		ext = ".json"
	case "chunks":
		ext = ".jsonl"
	default:
		ext = ".bin"
	}

	filename := outputType + ext
	key := fmt.Sprintf("processed/%s/%s", docID, filename)

	// Upload to R2
	_, err := u.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
		Metadata: map[string]string{
			"output-type": outputType,
			"doc-id":      docID,
			"upload-time": time.Now().Format(time.RFC3339),
		},
	})

	if err != nil {
		u.logger.Error("Failed to upload processed output", zap.String("key", key), zap.Error(err))
		return "", fmt.Errorf("upload to R2: %w", err)
	}

	u.logger.Info("Processed output uploaded",
		zap.String("key", key),
		zap.String("doc_id", docID),
		zap.String("output_type", outputType),
		zap.Int("size", len(data)))

	return key, nil
}

// DownloadOriginal downloads an original document from R2
func (u *R2Uploader) DownloadOriginal(ctx context.Context, docID string, filename string) ([]byte, error) {
	key := fmt.Sprintf("originals/%s/%s", docID, filename)

	result, err := u.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		u.logger.Error("Failed to download original", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("download from R2: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		u.logger.Error("Failed to read R2 response", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("read response: %w", err)
	}

	u.logger.Info("Original document downloaded",
		zap.String("key", key),
		zap.String("doc_id", docID),
		zap.Int("size", len(data)))

	return data, nil
}

// DownloadProcessed downloads processed output from R2
func (u *R2Uploader) DownloadProcessed(ctx context.Context, docID string, outputType string) ([]byte, error) {
	ext := ""
	switch outputType {
	case "docling_full", "mistral_ocr", "metadata", "docling_structure", "docling_tables":
		ext = ".json"
	case "docling_markdown":
		ext = ".md"
	case "chunks":
		ext = ".jsonl"
	}

	filename := outputType + ext
	key := fmt.Sprintf("processed/%s/%s", docID, filename)

	result, err := u.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		u.logger.Error("Failed to download processed output", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("download from R2: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		u.logger.Error("Failed to read R2 response", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("read response: %w", err)
	}

	u.logger.Info("Processed output downloaded",
		zap.String("key", key),
		zap.String("doc_id", docID),
		zap.String("output_type", outputType),
		zap.Int("size", len(data)))

	return data, nil
}

// ListDocuments lists all documents in the originals bucket
func (u *R2Uploader) ListDocuments(ctx context.Context) ([]string, error) {
	var documents []string

	paginator := s3.NewListObjectsV2Paginator(u.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(u.bucket),
		Prefix: aws.String("originals/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			u.logger.Error("Failed to list documents", zap.Error(err))
			return nil, fmt.Errorf("list R2 objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				documents = append(documents, *obj.Key)
			}
		}
	}

	u.logger.Info("Listed documents", zap.Int("count", len(documents)))
	return documents, nil
}

// GenerateDocumentID generates a unique document ID
func GenerateDocumentID() string {
	return "doc_" + uuid.New().String()[:8] + uuid.New().String()[24:]
}
