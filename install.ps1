$ErrorActionPreference = "Stop"

# Detect architecture
$arch = "amd64" # Standard windows x64 amd64

$binaryName = "mean-windows-$arch.exe"
$downloadUrl = "https://github.com/mini-page/mean-cli/releases/latest/download/$binaryName"

$installDir = "$HOME\AppData\Local\Programs\mean"
If (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

$targetPath = "$installDir\mean.exe"

Write-Host "Installing mean-cli..."
Write-Host "Downloading from: $downloadUrl"

# Download binary
Invoke-RestMethod -Uri $downloadUrl -OutFile $targetPath

# Check if path is in environment
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -split ';' -notcontains $installDir) {
    Write-Host "Adding $installDir to User Path..."
    [Environment]::SetEnvironmentVariable("Path", $userPath + ";$installDir", "User")
    $env:Path += ";$installDir"
}

Write-Host ""
Write-Host "✓ mean-cli successfully installed at: $targetPath"
Write-Host "  Please restart your terminal and run 'mean' to launch the TUI or CLI!"
Write-Host ""
