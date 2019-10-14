// Package main launches the calculator example directly
package main

import (
	"bufio"
	"io/ioutil"
	"os"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/theme"
)

func main() {
	app := app.New()
	app.Settings().SetTheme(theme.LightTheme())
	iconFile, _ := os.Open("icon.png")
	iconData, _ := ioutil.ReadAll(bufio.NewReader(iconFile))
	app.SetIcon(fyne.NewStaticResource("icon", iconData))
	Show(app)
	app.Run()
}
