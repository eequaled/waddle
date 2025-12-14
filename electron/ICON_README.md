# App Icon

Place your app icon here as `icon.ico`.

## Requirements
- Format: ICO (Windows icon format)
- Recommended size: 256x256 pixels
- Should include multiple sizes: 16x16, 32x32, 48x48, 64x64, 128x128, 256x256

## Creating an Icon

### Option 1: Online Converter
1. Create a PNG image (256x256 or larger)
2. Use https://convertio.co/png-ico/ or https://icoconvert.com/
3. Save as `icon.ico` in this folder

### Option 2: Using ImageMagick
```bash
magick convert logo.png -define icon:auto-resize=256,128,64,48,32,16 icon.ico
```

### Option 3: Using GIMP
1. Open your image in GIMP
2. Scale to 256x256
3. Export as .ico (select multiple sizes)

## Temporary Solution
If no icon is provided, Electron will use a default icon.
The build will still work, but the app won't have a custom icon.
