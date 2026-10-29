package clients

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS3WithGarageIntegration(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	storage := GetTestGarage(t)

	path := "hello.txt"
	want := "Hello World!"

	err := storage.WriteObject(ctx, path, []byte(want), nil)
	require.NoError(t, err, "Failed to write object bucket %s, %v", storage.bucket, err)

	data, err := storage.ReadObject(ctx, path)
	require.NoError(t, err, "Failed to read object bucket %s, %v", storage.bucket, err)

	got := string(data)
	require.Equal(t, want, got, "they should be equal")
}
