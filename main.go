package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/JLugagne/forscadb/internal/dataforge"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/filecache"
)

//go:embed all:frontend/dist
var assets embed.FS

func cacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".cache", "forscadb")
}

func main() {
	svc := dataforge.Init()
	dir := cacheDir()

	app := NewApp(
		dir,
		svc.ConnCommands,
		svc.ConnQueries,
		svc.SQLIntrospect,
		svc.SQLCommands,
		svc.SQLQueries,
		svc.NoSQLCommands,
		svc.NoSQLQueries,
		svc.KVCommands,
		svc.KVQueries,
	)

	// Load saved window state
	width, height := 1280, 800
	startState := options.Normal
	savedState, err := filecache.LoadWindowState(dir)
	if err == nil && savedState.Width > 0 && savedState.Height > 0 {
		width = savedState.Width
		height = savedState.Height
		if savedState.Maximised {
			startState = options.Maximised
		}
	} else {
		// First launch: start maximised
		startState = options.Maximised
	}

	err = wails.Run(&options.App{
		Title:            "DataForge",
		Width:            width,
		Height:           height,
		MinWidth:         800,
		MinHeight:        500,
		WindowStartState: startState,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			// Restore saved position (only if not maximised)
			if savedState != nil && !savedState.Maximised && savedState.X >= 0 && savedState.Y >= 0 {
				wailsRuntime.WindowSetPosition(ctx, savedState.X, savedState.Y)
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			// Save window state before closing
			x, y := wailsRuntime.WindowGetPosition(ctx)
			w, h := wailsRuntime.WindowGetSize(ctx)
			maximised := wailsRuntime.WindowIsMaximised(ctx)
			_ = filecache.SaveWindowState(dir, filecache.WindowState{
				X:         x,
				Y:         y,
				Width:     w,
				Height:    h,
				Maximised: maximised,
			})
			return false // don't prevent close
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
