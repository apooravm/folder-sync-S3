package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	configMng "github.com/apooravm/folder-sync-S3/src/config"
	s3Sync "github.com/apooravm/folder-sync-S3/src/s3"

	"github.com/apooravm/folder-sync-S3/src/utils"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	s3Config      *configMng.S3_Config
	configName    = "s3Config.json"
	configPath    string
	targetPath    string
	LargeFileSize int64 = 100_1000_1000 // 100MB
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

	client := s3.NewFromConfig(cfg)

	s3Sync.Client = client
	s3Sync.S3Config = s3Config

	if len(os.Args) > 1 {
		if err := HandleCliArgs(); err != nil {
			log.Println(err.Error())
		}

		return
	}

	utils.ColourPrint("Bro what do you want 🤨", "cyan")
}

func HandleCliArgs() error {
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
			if err := s3Sync.DeleteFile(objectKeyToDelete); err != nil {
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
		if err := s3Sync.DownloadFile(objectKeyToDownload); err != nil {
			return err
		}

	case "list":
		objInfoSlice, err := s3Sync.GetObjectKeys()
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

		if err := s3Sync.UploadFile(targetToUpload); err != nil {
			return err
		}

	case "dir":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target path provided.")
		}

		dirPath := os.Args[2]
		if err := s3Sync.UploadDir(dirPath); err != nil {
			return err
		}

	default:
		fmt.Println("???")
	}

	return nil
}
