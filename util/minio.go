package util

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// Minio is a struct to wrap minio client
type Minio struct {
	Client *minio.Client
}

// NewMinio is a function to create a new Minio struct
func NewMinio(endpoint, accessKey, secretKey string, secure bool) (*Minio, error) {
	// Initialize minio client object.
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &Minio{Client: client}, nil
}

// UploadFolder uploads a local folder to the specified bucket in minio.
func (m *Minio) UploadFolder(ctx context.Context, bucketName, folderPath, minioPrefix string) (err error) {
	// Check if bucket exists and create if it doesn't exist
	found, err := m.Client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if !found {
		// 设置存储桶的访问控制列表 ACL
		policy := fmt.Sprintf(`{
			"Version":"2012-10-17",
			"Statement":[
				{
					"Action":["s3:GetObject"],
					"Effect":"Allow",
					"Principal":{"AWS":["*"]},
					"Resource":["arn:aws:s3:::%s/*"],
					"Sid":"",
					"Condition": {
						 "StringLike": {
							"aws:Referer": "*"
						}
					}
				}
			]
		}`, bucketName)
		err = m.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
		err = m.Client.SetBucketPolicy(ctx, bucketName, policy)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	strArr := strings.Split(filepath.ToSlash(folderPath), "/")
	fileName := strArr[1]
	fmt.Println("fileName", fileName)

	// Walk through local folder and upload all files
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk through local folder: %v", err)
		}

		// Ignore directories
		if info.IsDir() {
			return nil
		}

		// Prepare object name in MinIO
		objectName := filepath.ToSlash(strings.Replace(path, fileName, minioPrefix, 1))

		// Open file for reading
		//file, err := os.Open(path)
		//if err != nil {
		//	return fmt.Errorf("failed to open file %s: %v", path, err)
		//}
		//defer file.Close()

		var contentType string
		// 获取文件扩展名
		ext := filepath.Ext(path)
		fmt.Println("walk: path:", ext)
		if ext == "" {
			// 如果扩展名为空，使用默认的MIME类型
			contentType = "application/octet-stream"
		} else {
			// 根据扩展名获取MIME类型
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				// 如果无法获取MIME类型，使用默认的MIME类型
				contentType = "application/octet-stream"
			}
		}
		fmt.Println("walk: extesion:", contentType)
		fmt.Println("work objectName", objectName)
		// Upload object to MinIO
		_, err = m.Client.FPutObject(ctx, bucketName, objectName, path, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("failed to upload object %s to MinIO: %v", objectName, err)
		}

		return nil
	})

	return nil
}
