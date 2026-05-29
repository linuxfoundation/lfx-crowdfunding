// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var s3Tracer = otel.Tracer("s3-presign-client")

const defaultPresignExpiry = 3 * time.Minute

// S3PresignClient generates short-lived presigned PUT URLs for logo uploads.
type S3PresignClient interface {
	// PresignLogoUpload returns a presigned PUT URL (uploadURL), the permanent
	// public destination URL (destinationURL), and the headers (requiredHeaders)
	// that the client MUST include on the PUT request for the signature to
	// validate. The map is derived from the SDK's SignedHeader field.
	// contentType must be a valid MIME type (e.g. "image/png").
	PresignLogoUpload(ctx context.Context, contentType string) (uploadURL, destinationURL string, requiredHeaders map[string]string, err error)
}

// S3Config holds settings for the S3 presign client.
type S3Config struct {
	// BucketName is the S3 bucket used for logo uploads (required).
	BucketName string
	// Region is the AWS region hosting the bucket.
	// When empty the SDK resolves it from the environment (AWS_REGION / AWS_DEFAULT_REGION).
	Region string
	// PresignExpiry is how long the presigned PUT URL remains valid.
	// Defaults to 3 minutes if zero.
	PresignExpiry time.Duration
}

type s3PresignClient struct {
	presigner *s3.PresignClient
	cfg       S3Config
	region    string // resolved AWS region (used to build the destination URL)
}

// NewS3PresignClient creates an S3PresignClient using the default AWS credential
// chain (env vars, shared credentials file, IAM instance role, etc.).
func NewS3PresignClient(ctx context.Context, cfg S3Config) (S3PresignClient, error) {
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("s3: BucketName is required")
	}
	if cfg.PresignExpiry == 0 {
		cfg.PresignExpiry = defaultPresignExpiry
	}

	opts := []func(*awsconfig.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3: load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)
	return &s3PresignClient{
		presigner: s3.NewPresignClient(s3Client),
		cfg:       cfg,
		region:    awsCfg.Region,
	}, nil
}

// PresignLogoUpload generates a presigned PUT URL and the resulting public URL.
// A fresh UUID key is generated for every call so uploads never overwrite each other.
// The returned requiredHeaders map must be sent verbatim on the PUT request;
// omitting any of them will cause S3 to reject the upload with a signature error.
func (c *s3PresignClient) PresignLogoUpload(ctx context.Context, contentType string) (string, string, map[string]string, error) {
	ctx, span := s3Tracer.Start(ctx, "S3PresignClient.PresignLogoUpload")
	defer span.End()
	span.SetAttributes(
		attribute.String("s3.bucket", c.cfg.BucketName),
		attribute.String("s3.content_type", contentType),
	)

	key := uuid.New().String()

	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.cfg.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = c.cfg.PresignExpiry
	})
	if err != nil {
		span.RecordError(err)
		return "", "", nil, fmt.Errorf("s3: presign put object: %w", err)
	}

	// Flatten the SDK's SignedHeader map (multi-value → comma-joined) so the
	// caller has a simple map[string]string to return to the frontend.
	requiredHeaders := make(map[string]string, len(req.SignedHeader))
	for k, vs := range req.SignedHeader {
		requiredHeaders[k] = strings.Join(vs, ",")
	}

	// Use the regional endpoint so the URL works without an HTTP redirect.
	// awsCfg.Region is always populated after LoadDefaultConfig succeeds.
	destinationURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.cfg.BucketName, c.region, key)
	return req.URL, destinationURL, requiredHeaders, nil
}
