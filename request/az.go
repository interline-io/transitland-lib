package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Bucket = &Az{}
	var _ Presigner = &Az{}
}

type Az struct {
	Account   string
	Container string
	KeyPrefix string
}

func NewAzFromUrl(ustr string) (*Az, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	a := Az{Account: u.Host, Container: p[0], KeyPrefix: strings.Join(p[1:], "/")}
	return &a, nil
}

func (r *Az) SetSecret(secret dmfr.Secret) error {
	return nil
}

func (r Az) Download(ctx context.Context, key string) (io.ReadCloser, int, error) {
	if key == "" {
		return nil, 0, errors.New("key must not be empty")
	}
	// Create request
	_, blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return nil, 0, err
	}
	azKey := strings.TrimPrefix(key, "/")
	if r.KeyPrefix != "" {
		azKey = r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	}
	rs, err := blobClient.DownloadStream(ctx, r.Container, azKey, nil)
	return rs.Body, 0, err
}

func (r Az) DownloadAuth(ctx context.Context, key string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, key)
}

func (r Az) ListAll(ctx context.Context, prefix string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (r Az) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	if key == "" {
		return errors.New("key must not be empty")
	}
	_, blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return err
	}
	// Upload the file to the specified container and blob name
	azKey := strings.TrimPrefix(key, "/")
	if r.KeyPrefix != "" {
		azKey = r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	}
	_, err = blobClient.UploadStream(ctx, r.Container, azKey, uploadFile, nil)
	return err
}

func (r Az) CreateSignedUrl(ctx context.Context, key string, contentDisposition string) (string, error) {
	if key == "" {
		return "", errors.New("key must not be empty")
	}
	cred, _, err := getAzBlobClient(r.Account)
	if err != nil {
		return "", err
	}
	now := time.Now().In(time.UTC).Add(time.Second * -10)
	expiry := now.Add(1 * time.Hour)
	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s", r.Account),
		cred,
		&service.ClientOptions{},
	)
	if err != nil {
		return "", err
	}
	info := service.KeyInfo{
		Start:  to.Ptr(now.UTC().Format(sas.TimeFormat)),
		Expiry: to.Ptr(expiry.UTC().Format(sas.TimeFormat)),
	}
	udc, err := svcClient.GetUserDelegationCredential(ctx, info, nil)
	if err != nil {
		return "", err
	}

	// Create Blob Signature Values with desired permissions and sign with user delegation credential
	sasQueryParams, err := sas.BlobSignatureValues{
		ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, url.PathEscape(contentDisposition)),
		Protocol:           sas.ProtocolHTTPS,
		StartTime:          now,
		ExpiryTime:         expiry,
		Permissions:        to.Ptr(sas.ContainerPermissions{Read: true}).String(),
		BlobName:           key,
		ContainerName:      r.Container,
	}.SignWithUserDelegation(udc)
	if err != nil {
		return "", err
	}
	sasUrl := fmt.Sprintf("https://%s/%s/%s?%s", r.Account, r.Container, key, sasQueryParams.Encode())
	return sasUrl, nil
}

func getAzBlobClient(account string) (*azidentity.DefaultAzureCredential, *azblob.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, nil, err
	}
	accountUrl := fmt.Sprintf("https://%s", strings.TrimPrefix(account, "az://"))
	blobClient, err := azblob.NewClient(accountUrl, credential, nil)
	if err != nil {
		return nil, nil, err
	}
	return credential, blobClient, nil
}
