package minioX

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

var (
	ErrMinioServer = errors.New("minio server error")
	ErrFileUpload  = errors.New("file upload error")
	ErrGetFileURL  = errors.New("get file url error")
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

// CreateBucket 创建bucket
func (c *Client) CreateBucket(ctx context.Context, bucketName string) error {
	exists, err := c.ExistBucket(ctx, bucketName)
	if err != nil {
		return errors.Join(ErrMinioServer, err)
	}
	if !exists {
		return c.MakeBucket(ctx, bucketName)
	}
	return nil
}

// ExistBucket 判断bucket是否存在
func (c *Client) ExistBucket(ctx context.Context, bucketName string) (bool, error) {
	exists, err := c.intraConn.conn.BucketExists(ctx, bucketName)
	if err != nil {
		return false, errors.Join(ErrMinioServer, err)
	}
	return exists, nil
}

// UploadLocalFile 将本地文件上传至minio
func (c *Client) UploadLocalFile(
	ctx context.Context, filePath string, bucketName string, fileName string, opt minio.PutObjectOptions,
) error {
	uploadInfo, err := c.intraConn.conn.FPutObject(
		ctx, bucketName, fileName, filePath, opt)
	if err != nil {
		return errors.Join(ErrFileUpload, err)
	}
	fmt.Println("successfully uploaded object: ", uploadInfo)
	return nil
}

// UploadSizeFile 读取固定大小的文件并上传至minio(主要使用)
func (c *Client) UploadSizeFile(
	ctx context.Context, bucketName string, fileName string, reader io.Reader, size int64, opt minio.PutObjectOptions,
) error {
	uploadInfo, err := c.intraConn.conn.PutObject(ctx, bucketName, fileName, reader, size, opt)
	if err != nil {
		return errors.Join(ErrFileUpload, err)
	}
	fmt.Println("successfully uploaded bytes: ", uploadInfo)
	return nil
}

// GetFileURL 根据文件名从minio获取文件URL
func (c *Client) GetFileURL(ctx context.Context, bucketName string, fileName string, timeLimit time.Duration) (*url.URL, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content", "attachment; filename=\""+fileName+"\"")

	preSignedURL, err := c.extraConn.conn.PresignedGetObject(ctx, bucketName, fileName, timeLimit, reqParams)
	if err != nil {
		return nil, errors.Join(ErrGetFileURL, err)
	}
	fmt.Println("successfully generated url")
	return preSignedURL, nil
}

// MakeBucket 创建bucket
func (c *Client) MakeBucket(ctx context.Context, bucketName string) error {
	return c.intraConn.conn.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: "cn-south",
	})
}
