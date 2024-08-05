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
	if err := CheckAndCreateConfig(); err != nil {
		fmt.Println(err.Error())
	}

	if len(os.Args) > 1 {
		if err := HandleCliArgs(); err != nil {
			log.Println(err.Error())
		}

		return
	}
}

func ReadConfig() error {
	var err error
	s3Config, err = configMng.ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf("Could not read the foldersync config. If the file does not exist, try %s", err.Error())
	}

	return nil
}

// Bucket client setup
func BucketClientSetup() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		utils.LogColourPrint("red", true, "Error loading config.", err.Error())
		return fmt.Errorf("Could not load the config. %s", err.Error())
	}

	client := s3.NewFromConfig(cfg)

	s3Sync.Client = client
	s3Sync.S3Config = s3Config

	return nil
}

func PrintHelp() {
	fmt.Println("App usage: 'folder-sync-s3.exe [COMMAND] [CMD_ARG] | {SUB_CMD}'")

	fmt.Println("\nCommands -")
	fmt.Println("help - Display this helper text. 'tshare-client.exe help")
	fmt.Println("config - Manage local config.json file. 'folder-sync-s3.exe config [SUB_CMD]'")
	fmt.Println("      delete - Delete config file.")
	fmt.Println("      generate - Create a new config file.")

	fmt.Println("\nSubcommands - Attach these at the end")
	fmt.Println("Set a custom chunk size. '-chunk=<CHUNK_SIZE>'")
	fmt.Println("Set a custom client name. '-name=<NAME>'")
	fmt.Println("Set to dev mode. '-mode=dev'")
}

func CheckAndCreateConfig() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Could not locate the foldersync exec file. %s", err.Error())
	}

	exeDir := filepath.Dir(exePath)
	configPath = filepath.Join(exeDir, configName)

	if !configMng.CheckConfigFileExists(configPath) {
		if err := configMng.CreateConfigFile(configPath); err != nil {
			return fmt.Errorf("creating config file. %s", err.Error())
		}

		fmt.Println("A config file has been generated. Please fill out the details at", configPath)
		return nil

	} else {
		if err := ReadConfig(); err != nil {
			return err
		}

		if s3Config.Bucket_name == "" || s3Config.Bucket_region == "" {
			fmt.Println("Details missing in config file. Fill them out at", configPath)
			return nil
		}
	}

	return nil
}

func HandleCliArgs() error {
	cliCMD := os.Args[1]

	switch cliCMD {
	case "help":
		PrintHelp()

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

	default:
		// Create config file if not exist
		if err := BucketClientSetup(); err != nil {
			return err
		}

		fmt.Println(s3Config)

		if err := S3CRUDArgs(); err != nil {
			return err
		}
	}

	return nil
}

func S3CRUDArgs() error {
	switch os.Args[1] {
	case "delete":
		if len(os.Args) < 3 {
			return fmt.Errorf("No target object key provided.")
		}

		if err := BucketClientSetup(); err != nil {
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
