# Go Calc App
This is a [go/golang](https://golang.org) port of the very useful [Calc.app](https://apps.micw.org) using [fyne](https://github.com/fyne-io/fyne) and [govaluate](https://github.com/Knetic/govaluate).

### Usage
Download from releases, run, enter `help` to get some help. It is not signed, google for "open unsigned mac/win".

For syntax, see https://github.com/Knetic/govaluate/blob/master/MANUAL.md with the following modifications:

  * `^` is converted to `**` (power)
  * `1.3e-3` etc is converted to float64
  * You can assign variables `a=1+3` (if you don't, the result is assigned to `r<number>`), and use them `a*2`

float64 is used internally, use mathematica if you need arbitrary precision math: `1+1e-19-1 == 0`

### Screenshot
![screenshot](https://user-images.githubusercontent.com/3457813/202898836-c3b45602-a5e1-40fd-ab45-e361902c78ef.png)


### Build
See `.github/workflows/go.yml`.

### Uses
  * [fyne](https://github.com/fyne-io/fyne) as GUI
  * [govaluate](https://github.com/Knetic/govaluate) to evaluate expressions
  * [action-automatic-releases](https://github.com/marvinpinto/action-automatic-releases) to create cross-platform releases using github actions.
