package clients

import (
	"bytes"
	"context"
	"testing"
)

func TestS3Storage_WriteAndRead(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeS3Client()
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
