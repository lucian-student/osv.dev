package clients

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/garage"
)

var (
	garageContainer *garage.Container
	client          *s3.Client
	once            sync.Once
	initError       error
)

func generateUniqueBucketName(t *testing.T) string {
	safeName := strings.ToLower(strings.NewReplacer("/", "-", "_", "-", ".", "-").Replace(t.Name()))
	if len(safeName) > 20 {
		safeName = safeName[:20]
	}
	uniqueID := uuid.New().String()[:8]
	return fmt.Sprintf("t-%s-%s", safeName, uniqueID)
}

func GetTestGarage(t *testing.T) *S3Storage {
	t.Helper()
	ctx := t.Context()

	once.Do(func() {

		garageContainer, initError = garage.Run(ctx, "dxflrs/garage:v2.4.1")
		if initError != nil {
			return
		}

		endpoint, err := garageContainer.ConnectionString(ctx)
		if err != nil {
			initError = err
			return
		}

		client = s3.New(s3.Options{
			BaseEndpoint: aws.String(endpoint),
			Region:       garageContainer.Region(),
			Credentials:  credentials.NewStaticCredentialsProvider(garageContainer.AccessKey(), garageContainer.SecretKey(), ""),
			UsePathStyle: true,
		})

	})

	if initError != nil {
		t.Fatalf("Failed to init garage:  %v", initError)
	}

	bucketName := generateUniqueBucketName(t)

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err, "failed to create isolated test bucket")
	t.Cleanup(func() {
		_, _ = client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
	})

	return NewS3StorageWithClient(client, bucketName)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if garageContainer != nil {
		_ = garageContainer.Terminate(context.Background())
	}
	os.Exit(code)
}
