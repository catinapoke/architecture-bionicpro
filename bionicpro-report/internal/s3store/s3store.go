package s3store

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	client  *minio.Client
	bucket  string
	cdnBase string
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	CDNBase   string
	UseSSL    bool
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	s := &Store{
		client:  client,
		bucket:  cfg.Bucket,
		cdnBase: strings.TrimRight(cfg.CDNBase, "/"),
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}

	// Убираем подписанные url, чтобы nginx мог кешировать ответы
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["*"]},
      "Action": ["s3:GetObject"],
      "Resource": ["arn:aws:s3:::%s/*"]
    }
  ]
}`, cfg.Bucket)
	if err := client.SetBucketPolicy(ctx, cfg.Bucket, policy); err != nil {
		return nil, fmt.Errorf("bucket policy: %w", err)
	}

	return s, nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
		return false, nil
	}
	return false, err
}

func (s *Store) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *Store) CDNURL(key string) string {
	u, err := url.JoinPath(s.cdnBase, s.bucket, key)
	if err != nil {
		return s.cdnBase + "/" + s.bucket + "/" + key
	}
	return u
}
