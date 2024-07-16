package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DeleteFile(objectKeyToDelete string) error {
	ObjInfo, err := ObjectKeyExists(objectKeyToDelete)
	if err != nil {
		return err
	}

	if ObjInfo == nil {
		return fmt.Errorf("Object key does not exist in the bucket.")
	}

	_, err = Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &S3Config.Bucket_name,
		Key:    &objectKeyToDelete,
	})

	if err != nil {
		return fmt.Errorf("Error deleting object %s. %s", objectKeyToDelete, err.Error())
	}

	return nil
}
