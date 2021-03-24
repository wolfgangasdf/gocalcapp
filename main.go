package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

//go:generate go run gen.go

func main() {
	app := app.New()
	app.Settings().SetTheme(theme.LightTheme())
	app.SetIcon(fyne.NewStaticResource("icon", icon()))
	Show(app)
	app.Run()
}
