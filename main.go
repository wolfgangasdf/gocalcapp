package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/theme"
)

//go:generate go run gen.go

func main() {
	app := app.New()
	app.Settings().SetTheme(theme.LightTheme())
	app.SetIcon(fyne.NewStaticResource("icon", icon()))
	Show(app)
	app.Run()
}
