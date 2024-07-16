package s3

import (
	"github.com/apooravm/folder-sync-S3/src/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	LargeFileSize  int64 = 100_1000_1000 // 100MB
	BaseObjectPath       = "sync/"
	S3Config       *config.S3_Config
	Client         *s3.Client
)
