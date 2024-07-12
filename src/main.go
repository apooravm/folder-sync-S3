package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	configMng "github.com/apooravm/folder-sync-S3/src/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	S3Config   *configMng.S3_Config
	configName = "s3Config.json"
	configPath string
	targetPath string
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		log.Println("Error locating the exec file")
		return
	}

	exeDir := filepath.Dir(exePath)
	configPath = filepath.Join(exeDir, configName)

	localCfg, err := configMng.ReadConfig(configPath)
	if err != nil {
		log.Println("Error reading config.", err.Error())
	}

	if len(os.Args) > 1 {
		err := handleCliArgs(localCfg)
		if err != nil {
			log.Println(err.Error())
			return
		}
		return
	}

	fmt.Println("Bro what do you want 🤨")

}

// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/s3/actions/bucket_basics.go
func ListObjects(localCfg *configMng.S3_Config) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Println("Error loading default config")
		return
	}

	client := s3.NewFromConfig(cfg)

	res, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &localCfg.Bucket_name,
	})

	if err != nil {
		log.Println("Couldnt list objects", err.Error())
		return
	}

	for _, item := range res.Contents {
		fmt.Println(string(*item.Key))
	}
}

func UploadFile(localCfg *configMng.S3_Config, targetToUpload string) error {
	file, err := os.Open(targetToUpload)
	if err != nil {
		return err
	}

	defer file.Close()

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("Error loading default config. %s", err.Error())
	}

	client := s3.NewFromConfig(cfg)

	// TODO: Create object key from targetToUpload.
	// Fornow default to a certain object folder

	string_parts := strings.Split(targetToUpload, "\\")
	fmt.Println(string_parts)

	targetPathInfo, err := os.Stat(targetToUpload)
	if err != nil {
		log.Println(err.Error())
	}

	fmt.Println(targetPathInfo.Name())
	fmt.Println(targetPathInfo.Size())
	fmt.Println(targetPathInfo.IsDir())

	return nil
	// Key is the object key btw
	res, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &localCfg.Bucket_name,
		Key:    &targetToUpload,
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("Error uploading file %s. %s", targetToUpload, err.Error())
	}

	fmt.Println(res.ResultMetadata)

	return nil
}

func handleCliArgs(localCfg *configMng.S3_Config) error {
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

		S3Config = cfg

		DownloadAndSaveFile()

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

		if err = UploadFile(localCfg, fileToUpload); err != nil {
			return fmt.Errorf("Error uploading. %s", err.Error())
		}

	default:
		fmt.Println("???")
	}

	return nil
}

func DownloadAndSaveFile() {
	file, err := DownloadFile(S3Config.Bucket_name, S3Config.Bucket_sync_folder+"info.json", S3Config.Bucket_region)
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
