package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/webdav-gateway/internal/config"
)

type Service struct {
	client       *minio.Client
	config       *config.Config
	bucketPrefix string
}

func NewService(cfg *config.Config) (*Service, error) {
	client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Service{
		client:       client,
		config:       cfg,
		bucketPrefix: cfg.MinIO.BucketPrefix,
	}, nil
}

func (s *Service) getBucketName(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", s.bucketPrefix, userID.String())
}

func (s *Service) EnsureBucket(ctx context.Context, userID uuid.UUID) error {
	bucketName := s.getBucketName(userID)

	exists, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	return nil
}

func (s *Service) PutObject(ctx context.Context, userID uuid.UUID, objectPath string, reader io.Reader, size int64, contentType string) error {
	bucketName := s.getBucketName(userID)
	objectKey := s.normalizePath(objectPath)

	_, err := s.client.PutObject(ctx, bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func (s *Service) GetObject(ctx context.Context, userID uuid.UUID, objectPath string) (*minio.Object, error) {
	bucketName := s.getBucketName(userID)
	objectKey := s.normalizePath(objectPath)

	obj, err := s.client.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	return obj, nil
}

func (s *Service) StatObject(ctx context.Context, userID uuid.UUID, objectPath string) (*minio.ObjectInfo, error) {
	bucketName := s.getBucketName(userID)
	objectKey := s.normalizePath(objectPath)

	info, err := s.client.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("stat object: %w", err)
	}

	return &info, nil
}

func (s *Service) DeleteObject(ctx context.Context, userID uuid.UUID, objectPath string) error {
	bucketName := s.getBucketName(userID)
	objectKey := s.normalizePath(objectPath)

	err := s.client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

func (s *Service) ListObjects(ctx context.Context, userID uuid.UUID, prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	bucketName := s.getBucketName(userID)
	normalizedPrefix := s.normalizePath(prefix)

	opts := minio.ListObjectsOptions{
		Prefix:    normalizedPrefix,
		Recursive: recursive,
	}

	var objects []minio.ObjectInfo
	for object := range s.client.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("list objects: %w", object.Err)
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (s *Service) CopyObject(ctx context.Context, userID uuid.UUID, srcPath, dstPath string) error {
	bucketName := s.getBucketName(userID)
	srcKey := s.normalizePath(srcPath)
	dstKey := s.normalizePath(dstPath)

	src := minio.CopySrcOptions{
		Bucket: bucketName,
		Object: srcKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: bucketName,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("copy object: %w", err)
	}

	return nil
}

func (s *Service) MoveObject(ctx context.Context, userID uuid.UUID, srcPath, dstPath string) error {
	if err := s.CopyObject(ctx, userID, srcPath, dstPath); err != nil {
		return err
	}

	if err := s.DeleteObject(ctx, userID, srcPath); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateFolder(ctx context.Context, userID uuid.UUID, folderPath string) error {
	bucketName := s.getBucketName(userID)
	folderKey := s.normalizePath(folderPath)
	
	if !strings.HasSuffix(folderKey, "/") {
		folderKey += "/"
	}

	_, err := s.client.PutObject(ctx, bucketName, folderKey, strings.NewReader(""), 0, minio.PutObjectOptions{
		ContentType: "application/x-directory",
	})
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	return nil
}

func (s *Service) DeleteFolder(ctx context.Context, userID uuid.UUID, folderPath string) error {
	bucketName := s.getBucketName(userID)
	prefix := s.normalizePath(folderPath)
	
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		opts := minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}
		for object := range s.client.ListObjects(ctx, bucketName, opts) {
			if object.Err == nil {
				objectsCh <- object
			}
		}
	}()

	errCh := s.client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})
	for err := range errCh {
		if err.Err != nil {
			return fmt.Errorf("delete folder: %w", err.Err)
		}
	}

	return nil
}

func (s *Service) normalizePath(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	return p
}

func (s *Service) GetObjectSize(ctx context.Context, userID uuid.UUID, objectPath string) (int64, error) {
	info, err := s.StatObject(ctx, userID, objectPath)
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}