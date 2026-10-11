# Garage S3 Integration for OSV.dev - Simple Config Guide

## The Reality

- GCS is NOT S3-compatible - it is Google's proprietary API.
- Garage IS S3-compatible - it speaks the AWS S3 API.
- Currently osv.dev uses `cloud.google.com/go/storage` (GCS SDK).
- To use Garage, you need to replace that storage client with an AWS S3 SDK v2 client.

## Why this is not a giant refactor

The project already has a generic storage abstraction:

File: `go/osv/clients/cloudstorage.go`

```go
type CloudStorage interface {
    ReadObject(ctx context.Context, path string) ([]byte, error)
    ReadObjectAttrs(ctx context.Context, path string) (*Attrs, error)
    WriteObject(ctx context.Context, path string, data []byte, opts *WriteOptions) error
    WriteObjectStream(ctx context.Context, path string, r io.Reader, opts *WriteOptions) error
    Objects(ctx context.Context, prefix string) iter.Seq2[*Object, error]
    Close() error
}


Adding aws dependency:

cd go
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3


adding client go/osv/clients/s3_client.go

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

type S3Config struct {
    Endpoint  string
    AccessKey string
    SecretKey string
    Bucket    string
    Region    string
}

type S3Client struct {
    client *s3.Client
    bucket string
}

var _ CloudStorage = (*S3Client)(nil)

func NewS3Client(cfg S3Config) (*S3Client, error) {
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

    credProvider := credentials.NewStaticCredentialsProvider(
        cfg.AccessKey,
        cfg.SecretKey,
        "",
    )

    client := s3.New(s3.Options{
        BaseEndpoint: aws.String(cfg.Endpoint),
        Region:       cfg.Region,
        Credentials:  credProvider,
        UsePathStyle: true,
    })

    return &S3Client{
        client: client,
        bucket: cfg.Bucket,
    }, nil
}

func (s *S3Client) ReadObject(ctx context.Context, path string) ([]byte, error) {
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

func (s *S3Client) ReadObjectAttrs(ctx context.Context, path string) (*Attrs, error) {
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

    if result.CustomMetadata != nil {
        if customTime, ok := result.CustomMetadata["custom-time"]; ok {
            t, err := time.Parse(time.RFC3339, customTime)
            if err == nil {
                attrs.CustomTime = t
            }
        }
    }

    return attrs, nil
}

func (s *S3Client) WriteObject(ctx context.Context, path string, data []byte, opts *WriteOptions) error {
    input := &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(path),
        Body:   bytes.NewReader(data),
    }

    if opts != nil && opts.ContentType != "" {
        input.ContentType = aws.String(opts.ContentType)
    }

    if opts != nil && opts.CustomTime != nil {
        input.Metadata = map[string]string{
            "custom-time": opts.CustomTime.Format(time.RFC3339),
        }
    }

    _, err := s.client.PutObject(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to write object: %w", err)
    }

    return nil
}

func (s *S3Client) WriteObjectStream(ctx context.Context, path string, r io.Reader, opts *WriteOptions) error {
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

func (s *S3Client) Objects(ctx context.Context, prefix string) iter.Seq2[*Object, error] {
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

                obj := &Object{
                    Name: *obj.Key,
                    Attrs: Attrs{
                        Updated: *obj.LastModified,
                    },
                }

                if !yield(obj, nil) {
                    return
                }
            }
        }
    }
}

func (s *S3Client) Close() error {
    return nil
}

## updating go/cmd/api/main.go

// Determine storage backend
var blobStore clients.CloudStorage
var err error

if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
    // Garage / S3 path
    blobStore, err = clients.NewS3Client(clients.S3Config{
        Endpoint:  s3Endpoint,
        AccessKey: os.Getenv("S3_ACCESS_KEY"),
        SecretKey: os.Getenv("S3_SECRET_KEY"),
        Bucket:    os.Getenv("S3_BUCKET"),
        Region:    os.Getenv("S3_REGION"),
    })
    if err != nil {
        logger.ErrorContext(ctx, "Failed to create S3 storage client", slog.Any("error", err))
        return err
    }
} else {
    // Existing GCS path
    gcsClient, err := storage.NewClient(ctx)
    if err != nil {
        logger.ErrorContext(ctx, "Failed to create storage client", slog.Any("error", err))
        return err
    }
    defer gcsClient.Close()

    vulnBucket := os.Getenv("OSV_VULNERABILITIES_BUCKET")
    if vulnBucket == "" {
        logger.ErrorContext(ctx, "OSV_VULNERABILITIES_BUCKET environment variable is not set")
        return errors.New("OSV_VULNERABILITIES_BUCKET environment variable is not set")
    }

    blobStore = clients.NewGCSClient(gcsClient, vulnBucket)
}

vulnStore := db.NewVulnerabilityStore(db.VulnStoreConfig{
    Client:           dbClient,
    GCS:              blobStore,
    BatchTimeout:     batchTimeout,
    BatchMaxElements: batchMaxElements,
})

## adding env variables

export S3_ENDPOINT=http://localhost:3900
export S3_ACCESS_KEY=GKxxxxxxxxxxxxx
export S3_SECRET_KEY=your_secret_key
export S3_BUCKET=osv-vulnerabilities
export S3_REGION=us-east-1

## 

clients.NewGCSClient(gcsClient, vulnBucket)

## 

This is not a “copy whole system to S3” change. It is:

1 new storage implementation
1 env var check
a few service startup files selecting the right backend
That is the correct way to do Garage support in this codebase.

If you want, I can give you the exact patch for:

go/cmd/api/main.go
go/cmd/website/main.go
go/cmd/worker/main.go
go/cmd/recoverer/main.go
go/cmd/exporter/exporter.go
in raw markdown form, ready to paste.