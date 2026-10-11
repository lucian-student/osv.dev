// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clients

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config holds S3/Garage connection configuration
type S3Config struct {
	Endpoint  string // e.g., http://localhost:3900 for Garage
	AccessKey string // e.g., GKxxxxxxxxxxxxx for Garage
	SecretKey string
	Bucket    string
	Region    string // default: us-east-1
}

// S3Storage implements CloudStorage interface for S3-compatible backends
// Works with: Garage, AWS S3, Minio, DigitalOcean Spaces, Wasabi, etc.
type S3Storage struct {
	client *s3.Client
	bucket string
}

// Compile-time check that S3Storage implements CloudStorage
var _ CloudStorage = (*S3Storage)(nil)

// NewS3Storage creates a new S3 storage client for S3-compatible backends
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("S3 endpoint required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 credentials required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket required")
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	// Create static credentials provider
	credProvider := credentials.NewStaticCredentialsProvider(
		cfg.AccessKey,
		cfg.SecretKey,
		"", // no session token
	)

	// Create S3 client with custom endpoint for Garage/Minio compatibility
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.Endpoint),
		Region:       cfg.Region,
		Credentials:  credProvider,
		UsePathStyle: true, // Required for Garage and Minio
	})

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// ReadObject reads the raw contents of an object
func (s *S3Storage) ReadObject(ctx context.Context, path string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// ReadObjectAttrs reads metadata about an object
func (s *S3Storage) ReadObjectAttrs(ctx context.Context, path string) (*Attrs, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read object attrs: %w", err)
	}

	attrs := &Attrs{
		Updated: *result.LastModified,
	}

	// Extract custom-time from metadata if present
	if result.Metadata != nil {
		if customTime, ok := result.Metadata["custom-time"]; ok {
			t, err := time.Parse(time.RFC3339, customTime)
			if err == nil {
				attrs.CustomTime = t
			}
		}
	}

	return attrs, nil
}

// WriteObject writes a byte slice to an object
func (s *S3Storage) WriteObject(ctx context.Context, path string, data []byte, opts *WriteOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(data),
	}

	if opts != nil && opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	// Store custom time as metadata since S3 doesn't have native custom-time support
	if opts != nil && opts.CustomTime != nil {
		input.Metadata = map[string]string{
			"custom-time": opts.CustomTime.Format(time.RFC3339),
		}
	}

	// Note: S3 doesn't support generation-based preconditions like GCS
	// IfGenerationMatches is ignored for S3 backends
	if opts != nil && opts.IfGenerationMatches != nil {
		// Log would go here if needed
		// S3 doesn't have generation numbers like GCS
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to write object: %w", err)
	}

	return nil
}

// WriteObjectStream writes data from an io.Reader to an object
func (s *S3Storage) WriteObjectStream(ctx context.Context, path string, r io.Reader, opts *WriteOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   r,
	}

	if opts != nil && opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to write object stream: %w", err)
	}

	return nil
}

// Objects returns an iterator over objects matching a prefix
func (s *S3Storage) Objects(ctx context.Context, prefix string) iter.Seq2[*Object, error] {
	return func(yield func(*Object, error) bool) {
		paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
			Prefix: aws.String(prefix),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				yield(nil, fmt.Errorf("failed to list objects: %w", err))
				return
			}

			for _, obj := range page.Contents {
				if obj.Key == nil {
					continue
				}

				objData := &Object{
					Name: *obj.Key,
					Attrs: Attrs{
						Updated: *obj.LastModified,
					},
				}

				if !yield(objData, nil) {
					return
				}
			}
		}
	}
}

// Close closes the S3 client (no-op for AWS SDK v2)
func (s *S3Storage) Close() error {
	return nil
}
