package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// FileMetadata stores enterprise file governance details.
type FileMetadata struct {
	ID             string            `json:"id"`
	Bucket         string            `json:"bucket"`
	ObjectName     string            `json:"object_name"`
	OriginalName   string            `json:"original_name"`
	TenantID       string            `json:"tenant_id"`
	OwnerID        string            `json:"owner_id"`
	SourceApp      string            `json:"source_app"`
	AssociatedObj  string            `json:"associated_obj,omitempty"` // e.g. "pms:project:123"
	ContentType    string            `json:"content_type"`
	SizeBytes      int64             `json:"size_bytes"`
	SHA256Checksum string            `json:"sha256_checksum"`
	Classification string            `json:"classification"` // public | internal | confidential | restricted
	RetentionUntil *time.Time        `json:"retention_until,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Client manages enterprise storage operations.
type Client struct {
	minioClient *minio.Client
	endpoint    string
	defaultBkt  string
	useSSL      bool
}

// NewClient initializes a MinIO / S3 client with zero-default security.
func NewClient(endpoint, accessKey, secretKey, defaultBucket string, useSSL bool) (*Client, error) {
	if endpoint == "" {
		endpoint = os.Getenv("MINIO_ENDPOINT")
		if endpoint == "" {
			endpoint = "localhost:9000"
		}
	}
	if accessKey == "" {
		accessKey = os.Getenv("MINIO_ROOT_USER")
		if accessKey == "" {
			accessKey = os.Getenv("MINIO_ACCESS_KEY")
		}
	}
	if secretKey == "" {
		secretKey = os.Getenv("MINIO_ROOT_PASSWORD")
		if secretKey == "" {
			secretKey = os.Getenv("MINIO_SECRET_KEY")
		}
	}
	if defaultBucket == "" {
		defaultBucket = os.Getenv("MINIO_DEFAULT_BUCKET")
		if defaultBucket == "" {
			defaultBucket = "statgate-files"
		}
	}

	if (accessKey == "" || secretKey == "") && strings.EqualFold(os.Getenv("STATGATE_ENV"), "production") {
		return nil, errors.New("MinIO credentials are required in production mode (SG-SEC-ZERO-DEFAULT)")
	}

	mClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &Client{
		minioClient: mClient,
		endpoint:    endpoint,
		defaultBkt:  defaultBucket,
		useSSL:      useSSL,
	}, nil
}

// EnsureBucket creates the bucket if it does not already exist.
func (c *Client) EnsureBucket(ctx context.Context, bucketName string) error {
	if c.minioClient == nil {
		return errors.New("storage client not initialized")
	}
	if bucketName == "" {
		bucketName = c.defaultBkt
	}
	exists, err := c.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return c.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}
	return nil
}

// Upload streams a file to object storage and computes SHA-256 checksum automatically.
func (c *Client) Upload(ctx context.Context, meta FileMetadata, reader io.Reader) (*FileMetadata, error) {
	if c.minioClient == nil {
		return nil, errors.New("storage client not initialized")
	}
	if meta.ID == "" {
		meta.ID = uuid.New().String()
	}
	if meta.Bucket == "" {
		meta.Bucket = c.defaultBkt
	}
	if meta.ObjectName == "" {
		meta.ObjectName = fmt.Sprintf("%s/%s/%s", meta.TenantID, meta.SourceApp, meta.ID)
	}
	if meta.Classification == "" {
		meta.Classification = "internal"
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	// Calculate checksum while streaming to memory / temp
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	opts := minio.PutObjectOptions{
		ContentType: meta.ContentType,
		UserMetadata: map[string]string{
			"x-statgate-id":             meta.ID,
			"x-statgate-tenant":         meta.TenantID,
			"x-statgate-classification": meta.Classification,
			"x-statgate-source-app":     meta.SourceApp,
			"x-statgate-associated-obj": meta.AssociatedObj,
		},
	}

	uploadInfo, err := c.minioClient.PutObject(ctx, meta.Bucket, meta.ObjectName, teeReader, meta.SizeBytes, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object to storage: %w", err)
	}

	meta.SizeBytes = uploadInfo.Size
	meta.SHA256Checksum = hex.EncodeToString(hasher.Sum(nil))

	return &meta, nil
}

// Download retrieves a reader for the stored object.
func (c *Client) Download(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	if c.minioClient == nil {
		return nil, errors.New("storage client not initialized")
	}
	if bucketName == "" {
		bucketName = c.defaultBkt
	}
	return c.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}
