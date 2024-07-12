package config

import (
	"encoding/json"
	"os"
)

type S3_Config struct {
	Bucket_name        string `json:"bucket_name"`
	Bucket_region      string `json:"bucket_region"`
	Bucket_sync_folder string `json:"bucket_sync_folder"`

	Aws_access_key_id    string `json:"aws_access_key_id"`
	Aws_secret_acess_key string `json:"aws_secret_access_key"`
}

// Checks whether a config file already exists at the given path
func CheckConfigFileExists(configPath string) bool {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateConfigFile(configPath string) error {
	configInit := S3_Config{
		Bucket_name:          "",
		Bucket_region:        "",
		Bucket_sync_folder:   "",
		Aws_access_key_id:    "",
		Aws_secret_acess_key: "",
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}

	defer file.Close()

	jsonData, err := json.MarshalIndent(&configInit, "", "    ")
	if err != nil {
		return err
	}

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}

	return nil
}

func ReadConfig(configPath string) (*S3_Config, error) {
	var localConfig S3_Config

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	// byteArr, err := io.ReadAll(file)
	// if err != nil {
	// 	return nil, fmt.Errorf("Error with io.ReadAll %s", err.Error())
	// }
	//
	// if err = json.Unmarshal(byteArr, &localConfig); err != nil {
	// 	return nil, fmt.Errorf("error unmarshalling %s", err.Error())
	// }
	//
	// fmt.Println(localConfig)
	//
	// return nil, nil

	if err = json.NewDecoder(file).Decode(&localConfig); err != nil {
		return nil, err
	}

	return &localConfig, nil
}

func DeleteConfig(configPath string) error {
	if err := os.Remove(configPath); err != nil {
		return err
	}

	return nil
}
