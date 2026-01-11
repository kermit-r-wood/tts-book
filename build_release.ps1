# build_release.ps1
$ErrorActionPreference = "Stop"

Write-Host "Starting Build Process..." -ForegroundColor Cyan

# Define paths
$ScriptDir = $PSScriptRoot
$FrontendDir = Join-Path $ScriptDir "frontend"
$BackendDir = Join-Path $ScriptDir "backend"
$ServerCmdDir = Join-Path $BackendDir "cmd\server"
$EmbedDistDir = Join-Path $ServerCmdDir "dist"

# 1. Build Frontend
Write-Host "`n[1/4] Building Frontend..." -ForegroundColor Yellow
Push-Location $FrontendDir
try {
    npm install
    npm run build
}
finally {
    Pop-Location
}

# 2. Prepare for Embedding
Write-Host "`n[2/4] Copying Assets..." -ForegroundColor Yellow
if (Test-Path $EmbedDistDir) {
    Remove-Item $EmbedDistDir -Recurse -Force
}
# Copy dist content to backend/cmd/server/dist
Copy-Item (Join-Path $FrontendDir "dist") -Destination $EmbedDistDir -Recurse

# 3. Build Backend
Write-Host "`n[3/4] Building Backend..." -ForegroundColor Yellow
Push-Location $BackendDir
try {
    $ExeName = "TTS-Book.exe"
    # Ensure dependencies are tidy
    go mod tidy
    # Build
    go build -o $ExeName .\cmd\server\main.go
    
    # Move executable to root for easy access
    Move-Item $ExeName $ScriptDir -Force
}
finally {
    Pop-Location
}

# 4. Cleanup (Optional: Don't remove the embedded dist if you want to inspect it, but technically we could)
# Write-Host "`n[4/4] Cleaning up..." -ForegroundColor Yellow

Write-Host "`nBuild Complete! Executable is at: $(Join-Path $ScriptDir 'TTS-Book.exe')" -ForegroundColor Green
Write-Host "Double-click TTS-Book.exe to run." -ForegroundColor Green
