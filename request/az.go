package request

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Store = &Az{}
	var _ Presigner = &Az{}
}

type Az struct {
	Account      string
	Container    string
	KeyPrefix    string
	VerifyUpload bool
}

func NewAzFromUrl(ustr string) (*Az, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	a := Az{
		Account:      u.Host,
		Container:    p[0],
		KeyPrefix:    strings.Join(p[1:], "/"),
		VerifyUpload: u.Query().Get("verify_upload") == "true",
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

	// Calculate MD5 before upload if verification is enabled and reader is seekable.
	// Note: Azure SDK provides TransactionalValidation for server-side MD5 verification:
	//   &azblob.UploadStreamOptions{
	//       TransactionalValidation: blob.TransferValidationTypeMD5(contentMD5),
	//   }
	// However, this is broken for UploadStream because it uploads the file in chunks
	// and validates the MD5 against each chunk, not the whole file. We must
	// re-download and verify the MD5 ourselves.
	var uploadMD5 []byte
	if r.VerifyUpload {
		uploadMD5 = md5FromReader(uploadFile)
		if uploadMD5 != nil {
			log.Trace().Str("key", key).Str("md5", hex.EncodeToString(uploadMD5)).Msg("az upload: calculated MD5 before upload")
		}
	}

	// Upload the file; Azure SDK handles retries internally (configured in getAzBlobClient)
	_, err = client.UploadStream(ctx, r.Container, r.getFullKey(key), uploadFile, nil)
	if err != nil {
		return err
	}

	// Verify upload by downloading and comparing MD5 if we were able to calculate it
	if uploadMD5 != nil {
		log.Trace().Str("key", key).Msg("az upload: verifying upload by downloading and comparing MD5")
		downloadStream, _, err := r.Download(ctx, key)
		if err != nil {
			return fmt.Errorf("upload verification failed: could not download for verification: %w", err)
		}
		defer downloadStream.Close()

		downloadMD5 := md5FromStream(downloadStream)
		if downloadMD5 == nil {
			return errors.New("upload verification failed: could not calculate MD5 of downloaded file")
		}

		if !bytes.Equal(uploadMD5, downloadMD5) {
			return fmt.Errorf("upload verification failed: MD5 mismatch between uploaded (%s) and downloaded (%s) file", hex.EncodeToString(uploadMD5), hex.EncodeToString(downloadMD5))
		}
		log.Trace().Str("key", key).Str("md5", hex.EncodeToString(downloadMD5)).Msg("az upload: MD5 verification successful")
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
	// Configure client with explicit retry policy (3 retries with exponential backoff)
	client, err := azblob.NewClient(accountUrl, credential, &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Retry: policy.RetryOptions{
				MaxRetries:    3,
				RetryDelay:    4 * time.Second,
				MaxRetryDelay: 120 * time.Second,
			},
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return credential, client, nil
}
