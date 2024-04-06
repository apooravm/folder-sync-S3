package config

type S3_Config struct {
	Bucket_name string `json:"bucket_name"`
	Bucket_region string `json:"bucket_region"`
	Bucket_sync_folder string `json:"bucket_sync_folder"`

	Aws_access_key_id string `json:"aws_access_key_id"`
	Aws_secret_acess_key string `json:"aws_secret_access_key"`
}
