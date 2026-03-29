package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"nssm-plus/internal/wrapper"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Service wrapper mode: nssm-plus.exe service <ServiceName>
	if len(os.Args) >= 3 && os.Args[1] == "service" {
		serviceName := os.Args[2]
		log.SetFlags(log.Ldate | log.Ltime)
		if err := wrapper.Run(serviceName); err != nil {
			log.Fatalf("Service %s failed: %v", serviceName, err)
		}
		return
	}

	// GUI mode (default)
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "NSSM Plus - Windows Service Manager",
		Width:     1100,
		Height:    720,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}
