package server

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Store struct {
	Bucket   string
	Client   *s3.Client
	Uploader *manager.Uploader
}

func NewS3Store(region, endpoint, accessKey, secretKey, bucket string) (*S3Store, error) {
	var cfg aws.Config
	var err error
	if accessKey != "" && secretKey != "" {
		cfg, err = awscfg.LoadDefaultConfig(context.Background(), awscfg.WithRegion(region), awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	} else {
		cfg, err = awscfg.LoadDefaultConfig(context.Background(), awscfg.WithRegion(region))
	}
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		}
	})
	up := manager.NewUploader(client)
	return &S3Store{Bucket: bucket, Client: client, Uploader: up}, nil
}

func (s *S3Store) SaveSubmission(instanceID string, name string, data []byte) error {
	key := strings.TrimPrefix(filepath.Join(instanceID, name), "/")
	_, err := s.Uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
		Body:   strings.NewReader(string(data)),
	})
	if err != nil {
		return fmt.Errorf("s3 upload submission: %w", err)
	}
	return nil
}

func (s *S3Store) SaveFile(instanceID string, fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	key := strings.TrimPrefix(filepath.Join(instanceID, filepath.Base(fh.Filename)), "/")
	_, err = s.Uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
		Body:   io.NopCloser(f),
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload file: %w", err)
	}
	// Optionally record metadata to DB via SaveAttachmentToDB by caller
	return key, nil
}
