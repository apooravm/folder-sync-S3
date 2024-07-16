package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apooravm/folder-sync-S3/src/utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DownloadFile(fileObjectKey string) error {
	objInfo, err := ObjectKeyExists(fileObjectKey)
	if err != nil {
		return err
	}

	if objInfo == nil {
		return fmt.Errorf("Object key does not exist in the bucket.")
	}

	fileBaseName := filepath.Base(fileObjectKey)

	var userConfirmRes string
	fmt.Printf("Downloading file %s (%.2fMB)\nContinue? (y/n)\n", objInfo.ObjectKey, float64(objInfo.Size)/float64(1000_000))
	fmt.Scan(&userConfirmRes)

	if userConfirmRes == "y" || userConfirmRes == "yes" || userConfirmRes == "Yes" || userConfirmRes == "Y" {
		// Depending on the size of file, choose the upload method
		// Right now a large file is ~100MB
		if objInfo.Size <= LargeFileSize {
			if err = DownloadNormalFile(fileObjectKey, fileBaseName); err != nil {
				return fmt.Errorf("Could not download. %s", err.Error())
			}

		} else {
			if err = DownloadLargeFile(fileObjectKey, fileBaseName); err != nil {
				return fmt.Errorf("Could not download. %s", err.Error())
			}
		}
		utils.ColourPrint("File downloaded successfully", "green")

	} else {
		fmt.Println(utils.LogColourSprintf("Aborted", "red", false))
	}

	return nil
}

func DownloadNormalFile(fileObjectKey string, fileBaseName string) error {
	output, err := Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &S3Config.Bucket_name,
		Key:    &fileObjectKey,
	})

	if err != nil {
		return err
	}

	defer output.Body.Close()

	file, err := os.Create("./downloads/" + fileBaseName)
	if err != nil {
		return fmt.Errorf("Could not create file. %s", err.Error())
	}

	defer file.Close()
	body, err := io.ReadAll(output.Body)
	if err != nil {
		return fmt.Errorf("Could not read object output. %s", err.Error())
	}

	_, err = file.Write(body)

	return nil
}

func DownloadLargeFile(fileObjectKey string, fileBaseName string) error {
	// TODO
	return nil
}
