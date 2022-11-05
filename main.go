package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:generate go run gen.go

func main() {
	app := app.New()
	app.SetIcon(fyne.NewStaticResource("icon", icon()))
	Show(app)
	app.Run()
}
