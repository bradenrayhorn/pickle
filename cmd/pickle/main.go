package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bradenrayhorn/pickle/bucket"
	"github.com/bradenrayhorn/pickle/connection"
	"github.com/bradenrayhorn/pickle/s3"
)

func main() {
	maintainCmd := flag.NewFlagSet("maintain", flag.ExitOnError)
	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)

	// Check if a command was provided
	if len(os.Args) < 2 {
		fmt.Println("Expected 'maintain' or 'backup' command")
		os.Exit(1)
	}

	// Parse the command
	switch os.Args[1] {
	case "maintain":
		err := maintainCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		config, _, err := loadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		bucket, err := bucket.New(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := bucket.RunMaintenance(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	case "backup":
		err := backupCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, s3config, err := loadConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		backupTargetConfig := loadBackupTargetConfig()

		if err := bucket.BackupBucket(s3config, backupTargetConfig); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Expected 'maintain' or 'backup' command")
		os.Exit(1)
	}
}

func loadConfig() (*bucket.Config, s3.Config, error) {
	conn, err := connection.FromString(os.Getenv("PICKLE_CONNECTION_CONFIG"))
	if err != nil {
		return nil, s3.Config{}, err
	}

	s3config := s3.Config{
		URL:          conn.URL,
		Region:       conn.Region,
		Bucket:       conn.Bucket,
		KeyID:        conn.KeyID,
		KeySecret:    conn.KeySecret,
		StorageClass: conn.StorageClass,
	}

	return &bucket.Config{
		Client:          s3.NewClient(s3config),
		ObjectLockHours: conn.ObjectLockHours,
	}, s3config, nil
}

func loadBackupTargetConfig() s3.Config {
	config := s3.Config{
		URL:          os.Getenv("PICKLE_BACKUP_S3_URL"),
		Region:       os.Getenv("PICKLE_BACKUP_S3_REGION"),
		KeyID:        os.Getenv("PICKLE_BACKUP_S3_KEY_ID"),
		KeySecret:    os.Getenv("PICKLE_BACKUP_S3_KEY_SECRET"),
		Bucket:       os.Getenv("PICKLE_BACKUP_S3_BUCKET"),
		StorageClass: os.Getenv("PICKLE_BACKUP_S3_STORAGE_CLASS"),
	}

	return config
}
