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
)

var (
	S3Config   *configMng.S3_Config
	configName = "s3Config.json"
	configPath string
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		log.Println("Error locating the exec file")
		return
	}

	exeDir := filepath.Dir(exePath)
	configPath = filepath.Join(exeDir, configName)

	cli_args := strings.Join(os.Args[1:], "")
	if len(cli_args) != 0 {
		handleCliArgs(cli_args)
		return
	}
}

func handleCliArgs(cliArg string) {
	switch cliArg {
	case "help":
		fmt.Println("Usage: `tjournal.exe [ARG]` if arg needed\n\nAvailable Args\nhelp   - Display help\ndelete - Delete user config.json")

	case "delete":
		if configMng.CheckConfigFileExists(configPath) {
			if err := configMng.DeleteConfig(configPath); err != nil {
				fmt.Println("Error deleting config")
				return

			} else {
				fmt.Println("Deleted successfully!")
				return
			}
		} else {
			fmt.Println("Config file does not exist")
		}
	case "generate":
		if configMng.CheckConfigFileExists(configPath) {
			fmt.Println("A Config already exists at path. Try 'folderSync.exe delete' to delete it.")
			return
		} else {
			if err := configMng.CreateConfigFile(configPath); err != nil {
				log.Println("Error creating config file", err.Error())
				return
			}
			fmt.Println("File created successfully. Please fill out the details.")
		}
	}
}

func DownloadAndSaveFile() {
	BUCKET_NAME := ""
	objKey := ""
	BUCKET_REGION := ""

	file, err := DownloadFile(BUCKET_NAME, objKey, BUCKET_REGION)
	if err != nil {
		log.Println("Error downloading file")
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
