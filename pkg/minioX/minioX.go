package minioX

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// ExtraConn 外网连接返回文件Url
type ExtraConn struct {
	conn *minio.Client
}

// IntraConn 内网连接服务上传文件
type IntraConn struct {
	conn *minio.Client
}

func NewExtraConn(c *minio.Client) ExtraConn {
	return ExtraConn{conn: c}
}

func NewIntraConn(c *minio.Client) IntraConn {
	return IntraConn{conn: c}
}

type Client struct {
	extraConn ExtraConn
	intraConn IntraConn
}

func NewClient(extraConn ExtraConn, intraConn IntraConn) *Client {
	return &Client{
		extraConn: extraConn,
		intraConn: intraConn,
	}
}

// UploadLocalFile 将本地文件上传至minio
func (c *Client) UploadLocalFile(ctx context.Context, filePath string, bucketName string, fileName string, opt minio.PutObjectOptions) error {
	// check whether the bucket exists
	if exists, err := c.intraConn.conn.BucketExists(ctx, bucketName); !(err == nil && exists) {
		return fmt.Errorf("minio buckect %s miss,err: %v\n", bucketName, err)
	}
	uploadInfo, err := c.intraConn.conn.FPutObject(
		ctx, bucketName, fileName, filePath, opt)
	if err != nil {
		return fmt.Errorf("failed uploaded object, err : %w", err)
	}
	fmt.Println("Successfully uploaded object: ", uploadInfo)
	return nil
}

// UploadSizeFile 读取固定大小的文件并上传至minio(主要使用)
func (c *Client) UploadSizeFile(ctx context.Context, bucketName string, fileName string, reader io.Reader, size int64, opt minio.PutObjectOptions) error {
	if exists, err := c.intraConn.conn.BucketExists(ctx, bucketName); !(err == nil && exists) {
		return fmt.Errorf("minio buckect %s miss,err: %v\n", bucketName, err)
	}
	uploadInfo, err := c.intraConn.conn.PutObject(ctx, bucketName, fileName, reader, size, opt)
	if err != nil {
		return fmt.Errorf("failed uploaded bytes, err : %w", err)
	}
	fmt.Println("Successfully uploaded bytes: ", uploadInfo)
	return nil
}

// GetFileURL 根据文件名从minio获取文件URL
func (c *Client) GetFileURL(ctx context.Context, bucketName string, fileName string, timeLimit time.Duration) (*url.URL, error) {
	if exists, err := c.extraConn.conn.BucketExists(ctx, bucketName); !(err == nil && exists) {
		return nil, fmt.Errorf("minio buckect %s miss,err: %v\n", bucketName, err)
	}
	reqParams := make(url.Values)
	reqParams.Set("response-content", "attachment; filename=\""+fileName+"\"")

	preSignedURL, err := c.extraConn.conn.PresignedGetObject(ctx, bucketName, fileName, timeLimit, reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed generated presigned URL, err : %w", err)
	}
	fmt.Println("Successfully generated preSigned URL", preSignedURL)
	return preSignedURL, nil
}
