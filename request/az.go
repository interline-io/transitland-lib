package request

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/avast/retry-go/v4"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Store = &Az{}
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
	a := Az{
		Account:   u.Host,
		Container: p[0],
		KeyPrefix: strings.Join(p[1:], "/"),
	}
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
	_, client, err := getAzBlobClient(r.Account)
	if err != nil {
		return nil, 0, err
	}
	rs, err := client.DownloadStream(ctx, r.Container, r.getFullKey(key), nil)
	return rs.Body, 0, err
}

func (r Az) DownloadAuth(ctx context.Context, key string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, key)
}

func (r Az) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	// List the containers in the storage account with a prefix
	_, client, err := getAzBlobClient(r.Account)
	if err != nil {
		return nil, err
	}
	// List the blobs in the container with a prefix
	pager := client.NewListBlobsFlatPager(r.Container, &azblob.ListBlobsFlatOptions{
		Prefix: to.Ptr(r.getFullKey(prefix)),
	})
	var ret []string
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, blob := range resp.Segment.BlobItems {
			ret = append(ret, stripDir(r.KeyPrefix, *blob.Name))
		}
	}
	return ret, nil
}

func (r Az) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	if key == "" {
		return errors.New("key must not be empty")
	}
	_, client, err := getAzBlobClient(r.Account)
	if err != nil {
		return err
	}

	// Get file seeker for retry capability and calculate MD5 for server-side verification
	var fileSeeker io.ReadSeeker
	var contentMD5 []byte
	if file, ok := uploadFile.(*os.File); ok {
		fileSeeker = file
		// Calculate MD5 for server-side verification (Azure verifies this during upload)
		// This allows us to verify integrity without a second request
		if _, err := file.Seek(0, io.SeekStart); err == nil {
			md5Hash := md5.New()
			if _, err := io.Copy(md5Hash, file); err == nil {
				contentMD5 = md5Hash.Sum(nil)
			}
			file.Seek(0, io.SeekStart) // Reset for upload
		}
	}

	// Prepare upload options with Content-MD5 for server-side verification
	var uploadOptions *azblob.UploadStreamOptions
	if len(contentMD5) > 0 {
		uploadOptions = &azblob.UploadStreamOptions{
			TransactionalValidation: blob.TransferValidationTypeMD5(contentMD5),
		}
	}

	// Upload function - extracted to avoid duplication
	uploadFn := func() error {
		_, err := client.UploadStream(ctx, r.Container, r.getFullKey(key), uploadFile, uploadOptions)
		return err
	}

	// Upload with retry if file is seekable, single attempt otherwise
	if fileSeeker == nil {
		// Can't retry if we can't seek back to beginning - single attempt only
		return uploadFn()
	}

	// Retry upload up to 3 times for transient errors
	uploadErr := retry.Do(
		uploadFn,
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.OnRetry(func(n uint, err error) {
			// Reset file position for retry
			fileSeeker.Seek(0, io.SeekStart)
		}),
	)

	if uploadErr != nil {
		return fmt.Errorf("upload failed: %w", uploadErr)
	}

	return nil
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

func (r Az) getFullKey(key string) string {
	azKey := strings.TrimPrefix(key, "/")
	if r.KeyPrefix != "" {
		azKey = r.KeyPrefix + "/" + azKey
	}
	return azKey
}

func getAzBlobClient(account string) (*azidentity.DefaultAzureCredential, *azblob.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, nil, err
	}
	accountUrl := fmt.Sprintf("https://%s", strings.TrimPrefix(account, "az://"))
	client, err := azblob.NewClient(accountUrl, credential, nil)
	if err != nil {
		return nil, nil, err
	}
	return credential, client, nil
}
