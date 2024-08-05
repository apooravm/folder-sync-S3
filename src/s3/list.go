package s3

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/s3/actions/bucket_basics.go
func GetObjectKeys() (*[]FileInfo, error) {
	res, err := Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &S3Config.Bucket_name,
	})

	if err != nil {
		return nil, fmt.Errorf("Could not list objects. %s", err.Error())
	}

	var objectKeySlice []FileInfo
	for _, item := range res.Contents {
		// Skip keys that are empty dirs.
		// Only add keys pointing to a file.
		if string(*item.Key)[len(*item.Key)-1] == '/' {
			continue
		}
		fileBaseName := filepath.Base(string(*item.Key))

		objectKeySlice = append(objectKeySlice, FileInfo{
			Filepath:  "./downloads/" + fileBaseName,
			Size:      *item.Size,
			ObjectKey: string(*item.Key),
		})
	}

	return &objectKeySlice, nil
}

type FileInfo struct {
	Filepath  string
	Size      int64
	ObjectKey string
}

// Fetches objectkeys and checks whether the input one exists
func ObjectKeyExists(objectKeyToCheck string) (*FileInfo, error) {
	objectKeySlice, err := GetObjectKeys()
	if err != nil {
		return nil, err
	}

	for _, objKey := range *objectKeySlice {
		if objectKeyToCheck == objKey.ObjectKey {
			return &objKey, nil
		}
	}

	return nil, nil
}
