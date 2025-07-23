$ErrorActionPreference = "Stop"

# Extract version from go.mod
$line = (Select-String -Path "go.mod" -Pattern "github\.com/wailsapp/wails").Line
$version = ($line -split '\s+')[2]
Write-Host "Installing wails $version"

# Install wails
go install "github.com/wailsapp/wails/v2/cmd/wails@$version"
