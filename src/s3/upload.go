package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apooravm/folder-sync-S3/src/utils"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func UploadNormalFile(targetToUpload string, targetObjectKey string) error {
	file, err := os.Open(targetToUpload)
	if err != nil {
		return err
	}

	defer file.Close()

	// Key is the object key btw
	_, err = Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &S3Config.Bucket_name,
		Key:    &targetObjectKey,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Error uploading file %s. %s", targetToUpload, err.Error())
	}

	return nil
}

func UploadLargeFile(targetToUpload string, targetObjectKey string) error {
	file, err := os.Open(targetToUpload)
	if err != nil {
		return fmt.Errorf("Error opening file. %s", err.Error())
	}

	defer file.Close()

	var partMiBs int64 = 10
	uploader := manager.NewUploader(Client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
		u.Concurrency = 5 // Default is 5
	})

	uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &S3Config.Bucket_name,
		Key:    &targetObjectKey,
		Body:   file,
	})

	return nil
}

func UploadFile(fileToUpload, fileTargetObjectPath string) error {
	targetPathInfo, err := os.Stat(fileToUpload)
	if err != nil {
		return fmt.Errorf("Could not find target info. %s", err.Error())
	}

	// Dir upload handled in same func and arg
	if targetPathInfo.IsDir() {
		var userConfirmRes string
		fmt.Printf("Uploading folder %s\nContinue? (y/n)", targetPathInfo.Name())
		fmt.Scan(&userConfirmRes)

		if userConfirmRes == "y" || userConfirmRes == "yes" || userConfirmRes == "Yes" || userConfirmRes == "Y" {
			if err := UploadDir(fileToUpload); err != nil {
				return err
			}
		} else {
			fmt.Println(utils.LogColourSprintf("Aborted", "red", false))
		}

		return nil
	}

	// Creating object key from filename
	targetPathObjectKey := "public/folder_sync/" + targetPathInfo.Name()
	if fileTargetObjectPath != "" {
		targetPathObjectKey = fileTargetObjectPath
	}

	var userConfirmRes string
	fmt.Printf("Uploading file %s (%.2fMB) to %s\nContinue? (y/n)\n", targetPathInfo.Name(), float64(targetPathInfo.Size())/float64(1000_000), targetPathObjectKey)
	fmt.Scan(&userConfirmRes)

	if userConfirmRes == "y" || userConfirmRes == "yes" || userConfirmRes == "Yes" || userConfirmRes == "Y" {
		// Depending on the size of file, choose the upload method
		// Right now a large file is ~100MB
		if targetPathInfo.Size() <= LargeFileSize {
			if err = UploadNormalFile(fileToUpload, targetPathObjectKey); err != nil {
				return fmt.Errorf("Could not upload. %s", err.Error())
			}

		} else {
			if err = UploadLargeFile(fileToUpload, targetPathObjectKey); err != nil {
				return fmt.Errorf("Could not upload. %s", err.Error())
			}
		}
		utils.ColourPrint(fmt.Sprintf("File uploaded successfully %s", targetPathObjectKey), "green")
	} else {
		fmt.Println(utils.LogColourSprintf("Aborted", "red", false))
	}

	return nil
}

func UploadDir(pathToUpload string) error {
	var dirFileInfo []FileInfo
	folderName := filepath.Base(pathToUpload)

	filepath.Walk(pathToUpload, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Modify path to start from the targetFolder
			// Lots of string manipulation, could be slow
			path_parts := strings.Split(filepath.ToSlash(path), "/")

			splitIdx := 0
			for i := 0; i < len(path_parts); i++ {
				if path_parts[i] == folderName {
					splitIdx = i
					break
				}
			}

			absFilePath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("Could not get abs filepath. %s", err.Error())
			}

			pathRelativeToTargetFolder := strings.Join(path_parts[splitIdx:], "/")

			dirFileInfo = append(dirFileInfo, FileInfo{
				Filepath:  absFilePath,
				Size:      info.Size(),
				ObjectKey: BaseObjectPath + pathRelativeToTargetFolder,
			})
		}

		return nil
	})

	var wg sync.WaitGroup

	// Btw, this creates a buffer channel with a certain capacity
	// allowing a limited number of values to be sent to the channel without requiring a corresponding receive.
	errChan := make(chan error, len(dirFileInfo))

	for _, file := range dirFileInfo {
		wg.Add(1)

		// Passing in file pointer triggers some bug due to how goroutines handle variables
		// Bug makes only the last file be uploaded multiple, len(dirFileInfo) times.
		// GPT exp
		// The issue with only the last file being uploaded could be due to a problem with
		// how the goroutines are handling the file information.
		// Specifically, the loop variable file is being captured by the goroutine,
		// leading to a common concurrency bug in Go.
		// In Go, loop variables are reused across iterations,
		// so all goroutines end up using the same file value unless it's passed explicitly to the goroutine.

		go func(file FileInfo) {
			defer wg.Done()

			if file.Size >= LargeFileSize {
				if err := UploadLargeFile(file.Filepath, file.ObjectKey); err != nil {
					errChan <- fmt.Errorf("%s could not be uploaded. %w", file.Filepath, err)
				} else {
					utils.ColourPrint("Upload successful "+file.ObjectKey, "green")
				}
			} else {
				if err := UploadNormalFile(file.Filepath, file.ObjectKey); err != nil {
					errChan <- fmt.Errorf("%s could not be uploaded. %w", file.Filepath, err)
				} else {
					utils.ColourPrint("Upload successful "+file.ObjectKey, "green")
				}
			}
		}(file)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		utils.ColourPrint(err.Error(), "red")
	}

	return nil
}
