package request

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/interline-io/transitland-lib/tl"
)

type S3 struct{}

func (r S3) Download(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	// Parse url
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return nil, 0, errors.New("could not parse url")
	}
	// Create client
	var client *s3.Client
	if secret.AWSAccessKeyID != "" && secret.AWSSecretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(secret.AWSAccessKeyID, secret.AWSSecretAccessKey, "")),
		)
		if err != nil {
			return nil, 0, err
		}
		client = s3.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, 0, err
		}
		client = s3.NewFromConfig(cfg)
	}
	// Get object
	s3bucket := s3uri.Host
	s3key := strings.TrimPrefix(s3uri.Path, "/")
	s3obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
	})
	if err != nil {
		return nil, 0, err
	}
	return s3obj.Body, 0, nil
}

func (r S3) Upload(ctx context.Context, ustr string, secret tl.Secret, uploadFile io.Reader) error {
	s3uri, err := url.Parse(ustr)
	if err != nil {
		return errors.New("could not parse url")
	}
	// Create client
	var client *s3.Client
	if secret.AWSAccessKeyID != "" && secret.AWSSecretAccessKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(secret.AWSAccessKeyID, secret.AWSSecretAccessKey, "")),
		)
		if err != nil {
			return err
		}
		client = s3.NewFromConfig(cfg)
	} else {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}
		client = s3.NewFromConfig(cfg)
	}
	// Save object
	s3bucket := s3uri.Host
	s3key := strings.TrimPrefix(s3uri.Path, "/")
	result, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3bucket),
		Key:    aws.String(s3key),
		Body:   uploadFile,
	})
	_ = result
	return err
}
