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
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Downloader = &S3{}
	var _ DownloaderAll = &S3{}
	var _ Uploader = &S3{}
	var _ UploaderAll = &S3{}
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
}

func (r S3) Download(ctx context.Context, key string, secret dmfr.Secret, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	// Create client
	client, err := awsConfig(ctx, secret)
	if err != nil {
		return nil, 0, err
	}
	// Get object
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := strings.TrimPrefix(r.KeyPrefix+"/"+strings.TrimPrefix(key, "/"), "/")
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return nil, 0, err
	}
	return s3obj.Body, 0, nil
}

func (r S3) Upload(ctx context.Context, key string, secret dmfr.Secret, uploadFile io.Reader) error {
	// Create client
	client, err := awsConfig(ctx, secret)
	if err != nil {
		return err
	}
	// Save object
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := strings.TrimPrefix(r.KeyPrefix+"/"+strings.TrimPrefix(key, "/"), "/")
	result, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
		Body:   uploadFile,
	})
	_ = result
	return err
}

func (r S3) CreateSignedUrl(ctx context.Context, key string, contentDisposition string, secret dmfr.Secret) (string, error) {
	client, err := awsConfig(ctx, secret)
	if err != nil {
		return "", err
	}
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := strings.TrimPrefix(r.KeyPrefix+"/"+strings.TrimPrefix(key, "/"), "/")
	presignClient := s3.NewPresignClient(client)
	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(1 * time.Hour)
	})
	return request.URL, err
}

func (h *S3) DownloadAll(ctx context.Context, outDir string, secret dmfr.Secret, checkFile func(string) bool) ([]string, error) {
	s, err := awsConfig(ctx, secret)
	if err != nil {
		return nil, err
	}
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(h.Bucket),
	}
	var objects []types.Object
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
			if checkFile(*obj.Key) {
				objects = append(objects, obj)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	/////////
	var ret []string
	for _, obj := range objects {
		result, err := s.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &h.Bucket,
			Key:    obj.Key,
		})
		if err != nil {
			return nil, err
		}
		defer result.Body.Close()
		outfn := filepath.Join(outDir, *obj.Key)
		if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
			return nil, err
		}
		ret = append(ret, outfn)
		f, err := os.Create(outfn)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, _, err := copyTo(f, result.Body, 0); err != nil {
			return nil, nil
		}
	}
	return ret, nil
}

func (h *S3) UploadAll(ctx context.Context, srcDir string, secret dmfr.Secret, checkFile func(string) bool) error {
	fns, err := findFiles(srcDir, checkFile)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		key := filepath.Join("", stripDir(srcDir, fn))
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		if err := h.Upload(ctx, key, secret, f); err != nil {
			return err
		}
	}
	return nil
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
