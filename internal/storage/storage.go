// Package storage talks to the S3 bucket
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/matsfxapp/matsfx-backend/internal/config"
)

const checkTimeout = 5 * time.Second

type Client struct {
	s3     *s3.Client
	bucket string
}

func Open(ctx context.Context, cfg config.S3) (*Client, error) {
	c := &Client{
		s3: s3.New(s3.Options{
			BaseEndpoint: aws.String(cfg.Endpoint),
			Region:       cfg.Region,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			UsePathStyle: true,
			// not every S3 server supports default checksums
			RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
			ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		}),
		bucket: cfg.Bucket,
	}
	if err := c.CheckBucket(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) CheckBucket(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	if _, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)}); err != nil {
		return fmt.Errorf("check bucket %s: %w", c.bucket, err)
	}
	return nil
}
