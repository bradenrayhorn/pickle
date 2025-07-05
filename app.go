package main

import (
	"context"
	"fmt"
	"pickle/connection"
	"time"

	"filippo.io/age"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
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
}

func (a *App) ArchiveFile() error {
	time.Sleep(time.Second * 10)

	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a file to archive",
	})
	if err != nil {
		return err
	}

	fmt.Println("archiving " + file)
	return nil
}
