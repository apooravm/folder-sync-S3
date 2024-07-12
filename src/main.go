package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	configMng "github.com/apooravm/folder-sync-S3/src/config"
	"github.com/apooravm/folder-sync-S3/src/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	s3Config      *configMng.S3_Config
	configName    = "s3Config.json"
	configPath    string
	targetPath    string
	LargeFileSize int64 = 100_1000_1000 // 100MB
	client        *s3.Client
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
func GetObjectKeys() (*[]string, error) {
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

	var objectKeySlice []string
	for _, item := range res.Contents {
		objectKeySlice = append(objectKeySlice, string(*item.Key))
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

	if targetPathInfo.IsDir() {
		return fmt.Errorf("Target is not a file. Dir upload unavailable.")
	}

	// Creating object key from filename
	targetPathObjectKey := "public/folder_sync/" + targetPathInfo.Name()

	fmt.Printf("Uploading file %s (%.2fMB) to %s\n", targetPathInfo.Name(), float64(targetPathInfo.Size())/float64(1000_000), targetPathObjectKey)

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

	return nil
}

func UploadDir(pathToUpload string) error {
	var dirFilePaths []string
	fmt.Println(filepath.Base(pathToUpload))
	filepath.Walk(pathToUpload, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			dirFilePaths = append(dirFilePaths, path)
		}

		return nil
	})

	for _, path := range dirFilePaths {
		fmt.Println(path)
	}

	return nil
}

func ObjectKeyExists(objectKeyToCheck string) (bool, error) {
	objectKeySlice, err := GetObjectKeys()
	if err != nil {
		return false, err
	}

	for _, objKey := range *objectKeySlice {
		if objectKeyToCheck == objKey {
			return true, nil
		}
	}

	return false, nil
}

func DeleteFile(objectKeyToDelete string) error {
	exists, err := ObjectKeyExists(objectKeyToDelete)
	if err != nil {
		return err
	}

	if !exists {
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
		cfg, err := configMng.ReadConfig(configPath)
		if err != nil {
			return fmt.Errorf("Error reading config. %s", err.Error())
		}

		s3Config = cfg

		DownloadAndSaveFile()

	case "list":
		objSlice, err := GetObjectKeys()
		if err != nil {
			return err
		}

		for _, objKey := range *objSlice {
			fmt.Println(objKey)
		}

		// Uploads file not dir YET
	case "upload":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target path provided.")
		}

		fileToUpload := os.Args[2]
		fileToUpload, err := filepath.Abs(fileToUpload)
		if err != nil {
			return fmt.Errorf("Error finding file. %s", err.Error())
		}

		UploadFile(fileToUpload)

		fmt.Println("File uploaded successfully.")
		utils.ColourPrint("Uploaded successfully.", "green")

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

func DownloadAndSaveFile() {
	file, err := DownloadFile(s3Config.Bucket_name, "public/notes/"+"info.json", s3Config.Bucket_region)
	if err != nil {
		log.Println("Error downloading file", err.Error())
		return
	}

	createDirPath := "./syncFolder/"
	// Create the required path
	err = os.MkdirAll(createDirPath, os.ModePerm)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(createDirPath+"file.txt", file, 0644); err != nil {
		log.Println("Error writing to file")
		return
	}
}

func DownloadFile(bucketName string, objPath string, region string) ([]byte, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	fmt.Println("BName", bucketName)
	fmt.Println("ObPath", objPath)
	fmt.Println("Region", region)
	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objPath),
	})

	if err != nil {
		return nil, err
	}

	defer output.Body.Close()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
