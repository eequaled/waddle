@echo off
REM Waddle Build Script for Windows (Batch version)
REM ================================================

echo ========================================
echo   Waddle Build Script
echo ========================================
echo.

echo [1/5] Checking prerequisites...

REM Check Go
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo   X Go not found. Install from https://go.dev/dl/
    exit /b 1
)
echo   √ Go found

REM Check Node.js
where node >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo   X Node.js not found. Install from https://nodejs.org/
    exit /b 1
)
echo   √ Node.js found

echo.
echo [2/5] Building Go backend...
go mod download
set CGO_ENABLED=0
go build -ldflags="-s -w" -o waddle-backend.exe .
if not exist waddle-backend.exe (
    echo   X Backend build failed
    exit /b 1
)
echo   √ Backend built: waddle-backend.exe

echo.
echo [3/5] Building React frontend...
cd frontend
if not exist node_modules (
    echo   Installing dependencies...
    call npm install
)
echo   Building production bundle...
call npm run build
cd ..
if not exist frontend\dist\index.html (
    echo   X Frontend build failed
    exit /b 1
)
echo   √ Frontend built

echo.
echo [4/5] Setting up Electron...
cd electron
if not exist node_modules (
    echo   Installing Electron dependencies...
    call npm install
)
cd ..
echo   √ Electron ready

echo.
echo [5/5] Building installer...
cd electron
call npm run build:win
cd ..

echo.
echo ========================================
echo   Build Complete!
echo ========================================
echo.
echo Installer location: dist-electron\
echo.
pause
