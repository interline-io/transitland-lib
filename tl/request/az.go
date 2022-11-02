package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/interline-io/transitland-lib/tl"
)

type Az struct{}

func (r Az) Download(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	// Parse url
	blobUri, err := url.Parse(ustr)
	if err != nil {
		return nil, 0, err
	}
	// Always use https as scheme because the internal url might be az://
	accountUrl := fmt.Sprintf("https://%s", blobUri.Host)
	path := strings.Split(blobUri.Path, "/")
	if len(path) < 2 {
		return nil, 0, errors.New("url requires container name and blob name in path")
	}
	containerName := path[1]
	blobName := strings.Join(path[2:], "/")
	// fmt.Println("account:", accountUrl, "container:", containerName, "blob:", blobName)

	// Create request
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, 0, err
	}
	blobClient, err := azblob.NewClient(accountUrl, credential, nil)
	if err != nil {
		return nil, 0, err
	}
	rs, err := blobClient.DownloadStream(ctx, containerName, blobName, nil)
	return rs.Body, 0, err
}

func (r Az) Upload(ctx context.Context, ustr string, secret tl.Secret, uploadFile io.Reader) error {
	// Parse url
	blobUri, err := url.Parse(ustr)
	if err != nil {
		return err
	}
	// Always use https as scheme because the internal url might be az://
	accountUrl := fmt.Sprintf("https://%s", blobUri.Host)
	path := strings.Split(blobUri.Path, "/")
	if len(path) < 2 {
		return errors.New("url requires container name and blob name in path")
	}
	containerName := path[1]
	blobName := strings.Join(path[2:], "/")

	// Create request
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return err
	}
	blobClient, err := azblob.NewClient(accountUrl, credential, nil)
	if err != nil {
		return err
	}

	// Upload the file to the specified container and blob name
	_, err = blobClient.UploadStream(ctx, containerName, blobName, uploadFile, nil)
	return err
}
