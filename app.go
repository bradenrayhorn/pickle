package main

import (
	"context"
	"fmt"
	"github.com/bradenrayhorn/pickle/bucket"
	"github.com/bradenrayhorn/pickle/connection"
	"github.com/bradenrayhorn/pickle/s3"

	"filippo.io/age"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context

	bucket *bucket.Config
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	runtime.WindowSetMinSize(ctx, 640, 640)
	a.ctx = ctx
}

// Connection info
func (a *App) GenerateAgeKey() (string, error) {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	return key.String(), nil
}

func (a *App) CreateConnectionString(config connection.Config) (string, error) {
	return connection.ToString(config)
}

// File management
func (a *App) InitializeConnection(connectionString string) error {
	conn, err := connection.FromString(connectionString)
	if err != nil {
		return err
	}

	key, err := age.ParseX25519Identity(conn.AgePrivateKey)
	if err != nil {
		return fmt.Errorf("parse age identity: %w", err)
	}

	a.bucket = &bucket.Config{
		Client: s3.NewClient(s3.Config{
			URL:          conn.URL,
			Region:       conn.Region,
			Bucket:       conn.Bucket,
			KeyID:        conn.KeyID,
			KeySecret:    conn.KeySecret,
			StorageClass: conn.StorageClass,
		}),
		Key: key,
	}

	return nil
}

func (a *App) SelectFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a file to archive",
	})
	if err != nil {
		return "", fmt.Errorf("select file: %w", err)
	}

	return file, err
}

func (a *App) ListFiles() ([]bucket.BucketFile, error) {
	b, err := bucket.New(a.bucket)
	if err != nil {
		return nil, err
	}

	return b.GetFiles()
}

func (a *App) UploadFile(diskPath string, targetPath string) error {
	b, err := bucket.New(a.bucket)
	if err != nil {
		return err
	}

	return b.UploadFile(diskPath, targetPath)
}
