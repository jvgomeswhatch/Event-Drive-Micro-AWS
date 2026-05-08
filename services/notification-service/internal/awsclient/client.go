package awsclient

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func newCfg(ctx context.Context) aws.Config {
	cfg, _ := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_DEFAULT_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "test"), getenv("AWS_SECRET_ACCESS_KEY", "test"), "")),
	)
	return cfg
}

func NewDynamo(ctx context.Context) *dynamodb.Client {
	cfg := newCfg(ctx)
	opts := []func(*dynamodb.Options){}
	if ep := os.Getenv("LOCALSTACK_ENDPOINT"); ep != "" {
		opts = append(opts, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(ep) })
	}
	return dynamodb.NewFromConfig(cfg, opts...)
}

func NewSQS(ctx context.Context) *sqs.Client {
	cfg := newCfg(ctx)
	opts := []func(*sqs.Options){}
	if ep := os.Getenv("LOCALSTACK_ENDPOINT"); ep != "" {
		opts = append(opts, func(o *sqs.Options) { o.BaseEndpoint = aws.String(ep) })
	}
	return sqs.NewFromConfig(cfg, opts...)
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
