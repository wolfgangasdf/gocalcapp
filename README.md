# Go Calc App
This is a [go/golang](https://golang.org) port of the very useful [Calc.app](https://apps.micw.org) using [fyne](https://github.com/fyne-io/fyne) and [govaluate](https://github.com/Knetic/govaluate).

### Usage
Download from releases, run, enter `help` to get some help.

### Build
See `.travis.yml`.

### Limits

#### float64 is used internally, use mathematica if you need arbitrary precision math:

`1+1e-19-1 == 0`

`1+(1*10**(-19))-1 == 0`
