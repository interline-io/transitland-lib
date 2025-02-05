package request

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Store = &S3{}
	var _ Presigner = &S3{}
}

func trimSlash(v string) string {
	return strings.TrimPrefix(strings.TrimSuffix(v, "/"), "/")
}

func NewS3FromUrl(ustr string) (*S3, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	s := S3{
		Bucket:    trimSlash(strings.TrimPrefix(u.Host, "s3://")),
		KeyPrefix: trimSlash(u.Path),
	}
	return &s, nil
}

type S3 struct {
	Bucket    string
	KeyPrefix string
	secret    dmfr.Secret
}

func (r *S3) SetSecret(secret dmfr.Secret) error {
	r.secret = secret
	return nil
}

func (r S3) getFullKey(key string) string {
	if r.KeyPrefix != "" {
		return r.KeyPrefix + "/" + trimSlash(key)
	}
	return trimSlash(key)
}

func (r S3) Download(ctx context.Context, key string) (io.ReadCloser, int, error) {
	// Create client
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return nil, 0, err
	}
	// Get object
	log.Debug().Msgf("s3 store: downloading key '%s', full key is '%s'", key, r.getFullKey(key))
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(r.getFullKey(key)),
	})
	if err != nil {
		return nil, 0, err
	}
	return s3obj.Body, 0, nil
}

func (r S3) DownloadAuth(ctx context.Context, key string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, key)
}

func (r S3) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	s, err := awsConfig(ctx, r.secret)
	if err != nil {
		return nil, err
	}
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(r.Bucket),
		Prefix: aws.String(r.getFullKey(prefix)),
	}
	var downloadKeys []string
	var output *s3.ListObjectsV2Output
	objectPaginator := s3.NewListObjectsV2Paginator(s, input)
	for objectPaginator.HasMorePages() {
		output, err = objectPaginator.NextPage(ctx)
		if err != nil {
			var noBucket *types.NoSuchBucket
			if errors.As(err, &noBucket) {
				err = noBucket
			}
			break
		}
		for _, obj := range output.Contents {
			if obj.Key == nil {
				continue
			}
			downloadKey := stripDir(r.KeyPrefix, *obj.Key)
			downloadKeys = append(downloadKeys, downloadKey)
		}
	}
	if err != nil {
		return nil, err
	}
	return downloadKeys, nil
}

func (r S3) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	// Create client
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return err
	}
	// Save object
	log.Debug().Msgf("s3 store: uploading to key '%s', full key is '%s'", key, r.getFullKey(key))
	result, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(r.getFullKey(key)),
		Body:   uploadFile,
	})
	_ = result
	return err
}

func (r S3) CreateSignedUrl(ctx context.Context, key string, contentDisposition string) (string, error) {
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return "", err
	}
	presignClient := s3.NewPresignClient(client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(r.getFullKey(key)),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(1 * time.Hour)
	})
	return request.URL, err
}

func awsConfig(ctx context.Context, secret dmfr.Secret) (*s3.Client, error) {
	// Create client
	var client *s3.Client
	if secret.AWSAccessKeyID != "" && secret.AWSSecretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(secret.AWSAccessKeyID, secret.AWSSecretAccessKey, "")),
		)
		if err != nil {
			return nil, err
		}
		client = s3.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		client = s3.NewFromConfig(cfg)
	}
	return client, nil
}
