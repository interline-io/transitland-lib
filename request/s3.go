package request

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
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
	var _ Bucket = &S3{}
	var _ Presigner = &S3{}
}

func NewS3FromUrl(ustr string) (*S3, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	s := S3{Bucket: u.Host, KeyPrefix: u.Path}
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
	trimKey := strings.TrimPrefix(key, "/")
	trimPrefix := strings.TrimPrefix(r.KeyPrefix, "/")
	return "/" + trimPrefix + "/" + trimKey
}

func (r S3) Download(ctx context.Context, key string) (io.ReadCloser, int, error) {
	// Create client
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return nil, 0, err
	}
	// Get object
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := r.getFullKey(key)
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return nil, 0, err
	}
	return s3obj.Body, 0, nil
}

func (r S3) DownloadAuth(ctx context.Context, key string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, key)
}

func (r *S3) DownloadAll(ctx context.Context, outDir string, downloadPrefix string, checkFile func(string) bool) ([]string, error) {
	s, err := awsConfig(ctx, r.secret)
	if err != nil {
		return nil, err
	}
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(r.Bucket),
		Prefix: aws.String(r.getFullKey(downloadPrefix)),
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
			relKey := stripDir(r.KeyPrefix, *obj.Key)
			if checkFile(relKey) {
				downloadKeys = append(downloadKeys, relKey)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	/////////
	var ret []string
	for _, downloadKey := range downloadKeys {
		// Get the object again
		result, err := s.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(r.Bucket),
			Key:    aws.String(r.getFullKey(downloadKey)),
		})
		if err != nil {
			return nil, err
		}
		defer result.Body.Close()
		// Strip the prefix from the object key, make path in output directory
		outfn := filepath.Join(outDir, stripDir(downloadPrefix, downloadKey))
		// Create the directory if necessary
		if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
			return nil, err
		}
		// Save the file
		if err := copyToFile(ctx, result.Body, outfn); err != nil {
			return nil, err
		}
		ret = append(ret, outfn)
	}
	return ret, nil
}

func (r S3) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	// Create client
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return err
	}
	// Save object
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := r.getFullKey(key)
	log.Debug().Msgf("s3 store: uploading to key '%s', full key is '%s'", key, s3key)
	result, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
		Body:   uploadFile,
	})
	_ = result
	return err
}

func (h *S3) UploadAll(ctx context.Context, srcDir string, prefix string, checkFile func(string) bool) error {
	fns, err := findFiles(srcDir, checkFile)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()
		key := filepath.Join(prefix, stripDir(srcDir, fn))
		if err := h.Upload(ctx, key, f); err != nil {
			return err
		}
	}
	return nil
}

func (r S3) CreateSignedUrl(ctx context.Context, key string, contentDisposition string) (string, error) {
	client, err := awsConfig(ctx, r.secret)
	if err != nil {
		return "", err
	}
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := r.getFullKey(key)
	presignClient := s3.NewPresignClient(client)
	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
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
