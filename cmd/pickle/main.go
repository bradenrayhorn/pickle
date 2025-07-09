package main

import (
	"flag"
	"fmt"
	"os"

	"filippo.io/age"
	"github.com/bradenrayhorn/pickle/bucket"
	"github.com/bradenrayhorn/pickle/connection"
	"github.com/bradenrayhorn/pickle/s3"
)

func main() {
	maintainCmd := flag.NewFlagSet("maintain", flag.ExitOnError)

	// Check if a command was provided
	if len(os.Args) < 2 {
		fmt.Println("Expected 'maintain' command")
		os.Exit(1)
	}

	// Parse the command
	switch os.Args[1] {
	case "maintain":
		maintainCmd.Parse(os.Args[2:])

		config, err := loadConfig()
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
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Expected 'maintain' command")
		os.Exit(1)
	}
}

func loadConfig() (*bucket.Config, error) {
	conn, err := connection.FromString(os.Getenv("PICKLE_CONNECTION_CONFIG"))
	if err != nil {
		return nil, err
	}

	key, err := age.ParseX25519Identity(conn.AgePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse age identity: %w", err)
	}

	return &bucket.Config{
		Client: s3.NewClient(s3.Config{
			URL:          conn.URL,
			Region:       conn.Region,
			Bucket:       conn.Bucket,
			KeyID:        conn.KeyID,
			KeySecret:    conn.KeySecret,
			StorageClass: conn.StorageClass,
		}),
		Key:             key,
		ObjectLockHours: conn.ObjectLockHours,
	}, nil
}
