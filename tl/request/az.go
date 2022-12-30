package request

import (
	"context"
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

	"github.com/interline-io/transitland-lib/tl"
)

type Az struct {
	Account   string
	Container string
	KeyPrefix string
}

func (r Az) Download(ctx context.Context, key string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	// Create request
	_, blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return nil, 0, err
	}
	azKey := r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	rs, err := blobClient.DownloadStream(ctx, r.Container, azKey, nil)
	return rs.Body, 0, err
}

func (r Az) Upload(ctx context.Context, key string, secret tl.Secret, uploadFile io.Reader) error {
	_, blobClient, err := getAzBlobClient(r.Account)
	if err != nil {
		return err
	}
	// Upload the file to the specified container and blob name
	// fmt.Println("account:", r.Account, "container:", r.Container, "prefix:", r.KeyPrefix, "key:", key)
	azKey := r.KeyPrefix + "/" + strings.TrimPrefix(key, "/")
	_, err = blobClient.UploadStream(ctx, r.Container, azKey, uploadFile, nil)
	return err
}

func (r Az) CreateSignedUrl(ctx context.Context, key string, secret tl.Secret) (string, error) {
	// fmt.Println("account:", r.Account)
	// fmt.Println("container:", r.Container)
	// fmt.Println("key:", key)
	cred, _, err := getAzBlobClient(r.Account)
	now := time.Now().In(time.UTC).Add(time.Second * -10)
	expiry := now.Add(1 * time.Hour)
	svcClient, err := service.NewClient(
		fmt.Sprintf("https://%s", r.Account),
		cred,
		&service.ClientOptions{},
	)
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
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     now,
		ExpiryTime:    expiry,
		Permissions:   to.Ptr(sas.ContainerPermissions{Read: true}).String(),
		BlobName:      key,
		ContainerName: r.Container,
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

func NewAzFromUrl(ustr string) (*Az, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	a := Az{Account: u.Host, Container: p[0], KeyPrefix: strings.Join(p[1:], "/")}
	return &a, nil
}
