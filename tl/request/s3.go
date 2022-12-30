package request

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/interline-io/transitland-lib/tl"
)

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

func (r S3) Download(ctx context.Context, key string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	// Create client
	client, err := awsConfig(ctx, secret)
	if err != nil {
		return nil, 0, err
	}
	// Get object
	s3bucket := strings.TrimPrefix(r.Bucket, "s3://")
	s3key := strings.TrimPrefix(r.KeyPrefix+"/"+strings.TrimPrefix(key, "/"), "/")
	// fmt.Printf("s3 download: bucket '%s' key: '%s'\n", s3bucket, s3key)
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return nil, 0, err
	}
	return s3obj.Body, 0, nil
}

func (r S3) Upload(ctx context.Context, key string, secret tl.Secret, uploadFile io.Reader) error {
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

// func (presigner Presigner) GetObject(
// 	bucketName string, objectKey string, lifetimeSecs int64) (*v4.PresignedHTTPRequest, error) {
// 	request, err := presigner.PresignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
// 		Bucket: aws.String(bucketName),
// 		Key:    aws.String(objectKey),
// 	}, func(opts *s3.PresignOptions) {
// 		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
// 	})
// 	if err != nil {
// 		log.Printf("Couldn't get a presigned request to get %v:%v. Here's why: %v\n",
// 			bucketName, objectKey, err)
// 	}
// 	return request, err
// }

func (r S3) CreateSignedUrl(ctx context.Context, key string, secret tl.Secret) (string, error) {
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
	return request.URL, nil
}

func awsConfig(ctx context.Context, secret tl.Secret) (*s3.Client, error) {
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
