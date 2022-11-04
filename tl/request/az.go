package request

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/interline-io/transitland-lib/tl"
)

type Az struct {
	Account   string
	Container string
	KeyPrefix string
}

func (r Az) Download(ctx context.Context, key string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	// Create request
	blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return nil, 0, err
	}
	azKey := r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	rs, err := blobClient.DownloadStream(ctx, r.Container, azKey, nil)
	return rs.Body, 0, err
}

func (r Az) Upload(ctx context.Context, key string, secret tl.Secret, uploadFile io.Reader) error {
	blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return err
	}
	// Upload the file to the specified container and blob name
	fmt.Println("account:", r.Account, "container:", r.Container, "prefix:", r.KeyPrefix, "key:", key)
	azKey := r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	_, err = blobClient.UploadStream(ctx, r.Container, azKey, uploadFile, nil)
	return err
}

func getAzBlobClient(account string) (*azblob.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	accountUrl := fmt.Sprintf("https://%s", strings.TrimPrefix(account, "az://"))
	blobClient, err := azblob.NewClient(accountUrl, credential, nil)
	if err != nil {
		return nil, err
	}
	return blobClient, nil
}
