package clients

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// simple in-memory fake implementing necessary methods
type fakeS3Client struct {
	objects map[string][]byte
	meta    map[string]map[string]string
	now     time.Time
}

func newFakeS3Client() *fakeS3Client {
	return &fakeS3Client{
		objects: make(map[string][]byte),
		meta:    make(map[string]map[string]string),
		now:     time.Now(),
	}
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	key := aws.ToString(params.Key)
	data, ok := f.objects[key]
	if !ok {
		message := fmt.Sprintf("No such key %s!", key)
		code := "404"
		return nil, &types.NoSuchKey{Message: &message, ErrorCodeOverride: &code}
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(data)),
	}, nil
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	key := aws.ToString(params.Key)
	buf := new(bytes.Buffer)
	if params.Body != nil {
		_, err := io.Copy(buf, params.Body)
		if err != nil {
			return nil, err
		}
	}
	f.objects[key] = buf.Bytes()
	if params.Metadata != nil {
		f.meta[key] = params.Metadata
	}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	key := aws.ToString(params.Key)
	_, ok := f.objects[key]
	if !ok {
		message := fmt.Sprintf("No such key %s!", key)
		code := "404"
		return nil, &types.NoSuchKey{Message: &message, ErrorCodeOverride: &code}
	}
	return &s3.HeadObjectOutput{
		LastModified: aws.Time(f.now),
		Metadata:     f.meta[key],
	}, nil
}

// ListObjectsV2 and CreateBucket not needed for these unit tests; implement stubs if necessary.
func (f *fakeS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	prefix := aws.ToString(params.Prefix)
	contents := make([]types.Object, 0)
	for k := range f.objects {
		if prefix == "" || (len(prefix) <= len(k) && k[:len(prefix)] == prefix) {
			contents = append(contents, types.Object{
				Key:          aws.String(k),
				LastModified: aws.Time(f.now),
			})
		}
	}
	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

func (f *fakeS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	// no-op for fake
	return &s3.CreateBucketOutput{}, nil
}

func TestS3Storage_WriteAndRead(t *testing.T) {
	ctx := context.Background()
	fake := newFakeS3Client()
	storage := NewS3StorageWithClient(fake, "bucket")

	key := "foo/bar.txt"
	data := []byte("hello world")

	if err := storage.WriteObject(ctx, key, data, &WriteOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	got, err := storage.ReadObject(ctx, key)
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %q want %q", string(got), string(data))
	}

	attrs, err := storage.ReadObjectAttrs(ctx, key)
	if err != nil {
		t.Fatalf("ReadObjectAttrs failed: %v", err)
	}
	if attrs.Updated.IsZero() {
		t.Fatalf("expected non-zero Updated")
	}
}
