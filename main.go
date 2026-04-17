package main

import (
	"embed"
	"flag"
	"log"

	"waddle/pkg/infra/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 1. Parse flags
	dataDirFlag := flag.String("data-dir", "", "Path to data directory (default: ~/.waddle)")
	portFlag := flag.String("port", "8080", "API Server port")
	flag.Parse()

	// 2. Load Config
	cfg := config.DefaultConfig()
	if *dataDirFlag != "" {
		cfg.DataDir = *dataDirFlag
	}
	if *portFlag != "" {
		cfg.Port = *portFlag
	}

	// 3. Create an instance of the app structure
	app := NewApp(cfg)

	// 4. Create application with options
	err := wails.Run(&options.App{
		Title:  "Waddle v2",
		Width:  1400,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}
