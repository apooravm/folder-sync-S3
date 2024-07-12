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
func ListObjects() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("Error loading default config")
		return
	}

	client := s3.NewFromConfig(cfg)

	res, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &s3Config.Bucket_name,
	})

	if err != nil {
		log.Println("Couldnt list objects", err.Error())
		return
	}

	for _, item := range res.Contents {
		fmt.Println(string(*item.Key))
	}
}

func UploadFile(targetToUpload string) error {
	file, err := os.Open(targetToUpload)
	if err != nil {
		return err
	}

	defer file.Close()

	// Creating object key from filename
	targetPathInfo, err := os.Stat(targetToUpload)
	if err != nil {
		log.Println(err.Error())
	}

	if targetPathInfo.IsDir() {
		return fmt.Errorf("Target is not a file. Dir upload unavailable.")
	}

	if targetPathInfo.Size() >= LargeFileSize {
		return fmt.Errorf("Placeholder error. Implement large file upload here.")
	}

	targetPathObjectKey := "public/folder_sync/" + targetPathInfo.Name()

	fmt.Printf("Uploading file %s (%.2fMB) to %s\n", targetPathInfo.Name(), float64(targetPathInfo.Size())/float64(1000_000), targetPathObjectKey)

	// Key is the object key btw
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &s3Config.Bucket_name,
		Key:    &targetPathObjectKey,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Error uploading file %s. %s", targetToUpload, err.Error())
	}

	fmt.Println("File uploaded successfully.")

	return nil
}

func UploadLargeObject(targetToUpload string) error {
	_, err := os.Open(targetToUpload)
	if err != nil {
		return fmt.Errorf("Error opening file. %s", err.Error())
	}

	var partMiBs int64 = 10
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})

	uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s3Config.Bucket_name),
		Key:    aws.String(objectKey),
		Body:   largeBuffer,
	})

	return nil
}

func handleCliArgs() error {
	cliCMD := os.Args[1]

	switch cliCMD {
	case "help":
		fmt.Println("Usage: `tjournal.exe [ARG]` if arg needed\n\nAvailable Args\nhelp   - Display help\ndelete - Delete user config.json")

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

	case "download":
		cfg, err := configMng.ReadConfig(configPath)
		if err != nil {
			return fmt.Errorf("Error reading config. %s", err.Error())
		}

		s3Config = cfg

		DownloadAndSaveFile()

	case "list":
		ListObjects()

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

		if err = UploadFile(fileToUpload); err != nil {
			return fmt.Errorf("Error uploading. %s", err.Error())
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
