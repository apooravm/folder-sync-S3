package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	configMng "github.com/apooravm/folder-sync-S3/src/config"
	"github.com/apooravm/folder-sync-S3/src/utils"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	s3Config       *configMng.S3_Config
	configName     = "s3Config.json"
	configPath     string
	targetPath     string
	LargeFileSize  int64 = 100_1000_1000 // 100MB
	client         *s3.Client
	BaseObjectPath = "sync/"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		log.Println("Error locating the exec file")
		return
	}

	exeDir := filepath.Dir(exePath)
	configPath = filepath.Join(exeDir, configName)

	s3Config, err = configMng.ReadConfig(configPath)
	if err != nil {
		log.Println("Error reading config.", err.Error())
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		utils.LogColourPrint("red", true, "Error loading config.", err.Error())
		return
	}

	client = s3.NewFromConfig(cfg)

	if len(os.Args) > 1 {
		if err := handleCliArgs(); err != nil {
			log.Println(err.Error())
		}

		return
	}

	utils.ColourPrint("Bro what do you want 🤨", "cyan")
}

// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/s3/actions/bucket_basics.go
func GetObjectKeys() (*[]FileInfo, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("Error loading default config. %s", err.Error())
	}

	client := s3.NewFromConfig(cfg)

	res, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &s3Config.Bucket_name,
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

func UploadNormalFile(targetToUpload string, targetObjectKey string) error {
	file, err := os.Open(targetToUpload)
	if err != nil {
		return err
	}

	defer file.Close()

	// Key is the object key btw
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &s3Config.Bucket_name,
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
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
		u.Concurrency = 5 // Default is 5
	})

	uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: &s3Config.Bucket_name,
		Key:    &targetObjectKey,
		Body:   file,
	})

	return nil
}

func UploadFile(fileToUpload string) error {
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

type FileInfo struct {
	Filepath  string
	Size      int64
	ObjectKey string
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

func DeleteFile(objectKeyToDelete string) error {
	ObjInfo, err := ObjectKeyExists(objectKeyToDelete)
	if err != nil {
		return err
	}

	if ObjInfo == nil {
		return fmt.Errorf("Object key does not exist in the bucket.")
	}

	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &s3Config.Bucket_name,
		Key:    &objectKeyToDelete,
	})

	if err != nil {
		return fmt.Errorf("Error deleting object %s. %s", objectKeyToDelete, err.Error())
	}

	return nil
}

func handleCliArgs() error {
	cliCMD := os.Args[1]

	switch cliCMD {
	case "help":
		fmt.Println("Usage: `tjournal.exe [ARG]` if arg needed\n\nAvailable Args\nhelp   - Display help\ndelete - Delete user config.json")

	case "config":
		if len(os.Args) < 3 {
			return fmt.Errorf("Yeah what about the config??")
		}

		config_cmd := os.Args[2]

		switch config_cmd {
		case "delete":
			if configMng.CheckConfigFileExists(configPath) {
				if err := configMng.DeleteConfig(configPath); err != nil {
					return fmt.Errorf("Error deleting config. %s", err.Error())

				} else {
					fmt.Println("Deleted successfully!")
				}
			} else {
				return fmt.Errorf("Config file not found.")
			}

		case "generate":
			if configMng.CheckConfigFileExists(configPath) {
				return fmt.Errorf("A Config already exists at path. Try 'folderSync.exe delete' to delete it.")
			} else {
				if err := configMng.CreateConfigFile(configPath); err != nil {
					return fmt.Errorf("creating config file. %s", err.Error())
				}

				fmt.Println("File created successfully. Please fill out the details at " + configPath)
			}
		}

	case "delete":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target object key provided.")
		}

		objectKeyToDelete := os.Args[2]

		var userConfirmRes string
		fmt.Println("Are you sure about this? (y/n)")
		fmt.Scan(&userConfirmRes)

		if userConfirmRes == "y" || userConfirmRes == "yes" || userConfirmRes == "Yes" || userConfirmRes == "Y" {
			fmt.Println(utils.LogColourSprintf(fmt.Sprintf("Deleting %s...", objectKeyToDelete), "yellow", false))
			if err := DeleteFile(objectKeyToDelete); err != nil {
				return fmt.Errorf("%s", err.Error())
			}

			utils.ColourPrint("Deleted successfully.", "green")

		} else {
			fmt.Println(utils.LogColourSprintf("Aborted", "red", false))
		}

	case "generate":

	case "download":
		if err := os.MkdirAll("./downloads", os.ModePerm); err != nil {
			return fmt.Errorf("Could not create downloads folder. %s", err.Error())
		}

		if len(os.Args) < 3 {
			return fmt.Errorf("No object key provided.")
		}

		objectKeyToDownload := os.Args[2]
		if err := DownloadFile(objectKeyToDownload); err != nil {
			return err
		}

	case "list":
		objInfoSlice, err := GetObjectKeys()
		if err != nil {
			return err
		}

		for _, objInfo := range *objInfoSlice {
			fmt.Printf("%s %s\n", utils.LogColourSprintf(fmt.Sprintf("[%.2fMB]", float64(objInfo.Size)/float64(1000_000)), "yellow", false), objInfo.ObjectKey)
		}

		// Uploads file not dir YET
	case "upload":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target path provided.")
		}

		targetToUpload := os.Args[2]
		targetToUpload, err := filepath.Abs(targetToUpload)
		if err != nil {
			return fmt.Errorf("Error finding file. %s", err.Error())
		}

		if err := UploadFile(targetToUpload); err != nil {
			return err
		}

	case "dir":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target path provided.")
		}

		dirPath := os.Args[2]
		if err := UploadDir(dirPath); err != nil {
			return err
		}

	default:
		fmt.Println("???")
	}

	return nil
}

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
	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &s3Config.Bucket_name,
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
