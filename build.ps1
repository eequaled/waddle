# Waddle Build Script for Windows
# ================================
# This script builds the complete installable application

param(
    [switch]$SkipFrontend,
    [switch]$SkipBackend,
    [switch]$Portable,
    [switch]$Dev
)

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Waddle Build Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check prerequisites
Write-Host "[1/5] Checking prerequisites..." -ForegroundColor Yellow

# Check Go
try {
    $goVersion = go version
    Write-Host "  ✓ Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "  ✗ Go not found. Install from https://go.dev/dl/" -ForegroundColor Red
    exit 1
}

# Check Node.js
try {
    $nodeVersion = node --version
    Write-Host "  ✓ Node.js: $nodeVersion" -ForegroundColor Green
} catch {
    Write-Host "  ✗ Node.js not found. Install from https://nodejs.org/" -ForegroundColor Red
    exit 1
}

# Check npm
try {
    $npmVersion = npm --version
    Write-Host "  ✓ npm: $npmVersion" -ForegroundColor Green
} catch {
    Write-Host "  ✗ npm not found" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Build Backend
if (-not $SkipBackend) {
    Write-Host "[2/5] Building Go backend..." -ForegroundColor Yellow
    
    # Download dependencies
    Write-Host "  Downloading dependencies..."
    go mod download
    
    # Build with optimizations
    Write-Host "  Compiling waddle-backend.exe..."
    $env:CGO_ENABLED = "0"
    go build -ldflags="-s -w" -o waddle-backend.exe .
    
    if (Test-Path "waddle-backend.exe") {
        $size = (Get-Item "waddle-backend.exe").Length / 1MB
        Write-Host "  ✓ Backend built: waddle-backend.exe ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "  ✗ Backend build failed" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[2/5] Skipping backend build" -ForegroundColor Gray
}

Write-Host ""

# Build Frontend
if (-not $SkipFrontend) {
    Write-Host "[3/5] Building React frontend..." -ForegroundColor Yellow
    
    Push-Location frontend
    
    # Install dependencies if needed
    if (-not (Test-Path "node_modules")) {
        Write-Host "  Installing dependencies..."
        npm install
    }
    
    # Build production bundle
    Write-Host "  Building production bundle..."
    npm run build
    
    Pop-Location
    
    if (Test-Path "frontend/dist/index.html") {
        Write-Host "  ✓ Frontend built: frontend/dist/" -ForegroundColor Green
    } else {
        Write-Host "  ✗ Frontend build failed" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[3/5] Skipping frontend build" -ForegroundColor Gray
}

Write-Host ""

# Setup Electron
Write-Host "[4/5] Setting up Electron..." -ForegroundColor Yellow

Push-Location electron

# Install Electron dependencies if needed
if (-not (Test-Path "node_modules")) {
    Write-Host "  Installing Electron dependencies..."
    npm install
}

Pop-Location

Write-Host "  ✓ Electron ready" -ForegroundColor Green
Write-Host ""

# Build Installer
if (-not $Dev) {
    Write-Host "[5/5] Building installer..." -ForegroundColor Yellow
    
    Push-Location electron
    
    if ($Portable) {
        Write-Host "  Building portable version..."
        npm run build:portable
    } else {
        Write-Host "  Building installer..."
        npm run build:win
    }
    
    Pop-Location
    
    if (Test-Path "dist-electron") {
        Write-Host "  ✓ Installer created in dist-electron/" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Output files:" -ForegroundColor Cyan
        Get-ChildItem "dist-electron" -Filter "*.exe" | ForEach-Object {
            $size = $_.Length / 1MB
            Write-Host "    - $($_.Name) ($([math]::Round($size, 2)) MB)"
        }
    } else {
        Write-Host "  ✗ Installer build failed" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[5/5] Skipping installer (dev mode)" -ForegroundColor Gray
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Build Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if ($Dev) {
    Write-Host "To run in development mode:" -ForegroundColor Yellow
    Write-Host "  1. Start backend: .\waddle-backend.exe"
    Write-Host "  2. Start frontend: cd frontend && npm run dev"
    Write-Host "  3. Start Electron: cd electron && npm start"
} else {
    Write-Host "Installer location: dist-electron\" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To install:" -ForegroundColor Yellow
    Write-Host "  Run the Waddle-*-Setup.exe installer"
    Write-Host ""
    Write-Host "For portable version:" -ForegroundColor Yellow
    Write-Host "  Run: .\build.ps1 -Portable"
}
