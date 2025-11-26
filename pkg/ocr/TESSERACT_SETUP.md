# Tesseract OCR Setup

## Quick Setup (Recommended)

1. Download Tesseract: https://github.com/UB-Mannheim/tesseract/wiki
2. Install to default location: `C:\Program Files\Tesseract-OCR\tesseract.exe`
3. Add English language data (included in installer)

## For Bundled Distribution

To bundle Tesseract with your app for distribution:

1. Download portable version or extract from installer
2. Copy `tesseract.exe` and `tessdata` folder to `pkg/ocr/bin/`
3. The app will automatically use the bundled version

## Resource Usage

- CPU: Moderate (per-image processing)
- Memory: ~50-100MB
- Disk: ~50MB (with English language pack)

## Language Support

Currently configured for English. To add more languages, download additional `.traineddata` files from:
https://github.com/tesseract-ocr/tessdata
