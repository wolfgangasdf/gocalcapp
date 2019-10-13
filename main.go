// Package main launches the calculator example directly
package main

import (
	"fmt"

	"fyne.io/fyne/app"
	"fyne.io/fyne/theme"
)

func main() {
	fmt.Println("huhu")

	app := app.New()
	app.Settings().SetTheme(theme.LightTheme())
	// app.SetIcon(icon.CalculatorBitmap)
	Show(app)
	app.Run()
}
