package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/Knetic/govaluate"
)

// https://developer.fyne.io/extend/numerical-entry.html

type inputField struct {
	*widget.Entry
	c *calc
}

func newInputField(c *calc) *inputField {
	e := &widget.Entry{} // don't use NewEntry https://github.com/fyne-io/fyne/issues/1624
	e.SetPlaceHolder(`Expression or "help"`)
	e.Wrapping = fyne.TextWrapBreak // this enables scrolling if too wide
	eif := &inputField{e, c}
	eif.ExtendBaseWidget(eif) // crucial for operation/focus https://github.com/fyne-io/fyne/issues/537
	return eif
}

type historyField struct {
	*widget.Entry
	c *calc
}

func newHistoryField(c *calc) *historyField {
	e := &widget.Entry{}
	ehf := &historyField{e, c}
	ehf.ExtendBaseWidget(ehf)
	return ehf
}

func (s *inputField) walkHistory(diff int) {
	s.c.inputHistPos += diff
	if s.c.inputHistPos < 0 || s.c.inputHistPos > len(s.c.inputHistory)-1 {
		s.c.inputHistPos = 0
	}
	if s.c.inputHistPos == 0 {
		s.mySetText("")
	} else {
		s.mySetText(s.c.inputHistory[len(s.c.inputHistory)-s.c.inputHistPos])
	}
}

func (s *inputField) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyReturn:
		s.c.evaluate()
	case fyne.KeyEscape:
		s.mySetText("")
	case fyne.KeyUp:
		s.walkHistory(1)
	case fyne.KeyDown:
		s.walkHistory(-1)
	default:
		s.Entry.TypedKey(key)
		s.c.window.Canvas().Refresh(s.c.window.Content()) // important, bug?
	}
}

func (s *inputField) mySetText(t string) {
	s.Entry.SetText(t)
	s.c.window.Canvas().Refresh(s.c.window.Content()) // important, bug?
	if t != "" {                                      // select all
		s.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
		s.Entry.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
		s.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
		s.Entry.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	}
}

func (e *historyField) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyEscape: // escape go back to input
		e.c.window.Canvas().Focus(e.c.input)
	default:
	}
}

func (e *historyField) TypedRune(r rune) {
	// ignore all input
}

func (e *historyField) TypedShortcut(shortcut fyne.Shortcut) {
	_, ok := shortcut.(*fyne.ShortcutCopy)
	if ok { // only allow copy, not paste
		e.Entry.TypedShortcut(shortcut)
		return
	}
}

type calc struct {
	input        *inputField
	inputHistory []string // array of entered inputs for cursur up/down history
	inputHistPos int
	history      *historyField
	scrollhist   *container.Scroll
	window       fyne.Window
	functions    map[string]govaluate.ExpressionFunction
	parameters   map[string]interface{} // variables
	configFile   string
	outformat    string
	lastR        int // highest used rXXX variable
	reResVar     *regexp.Regexp
}

func (c *calc) histScrollToEnd() {
	c.scrollhist.ScrollToBottom()
	// c.history.CursorRow = strings.Count(c.history.Text, "\n") + 1 // hack not needed anymore
}

func (c *calc) addToHistory(eres string) {
	c.history.SetText(c.history.Text + "\n" + eres)
	c.histScrollToEnd()
	c.inputHistPos = 0
}

func (c *calc) replaceexpnumbers(expr string) string {
	re := regexp.MustCompile(`([0-9]*\.?[0-9]+)[eE]([-+]?[0-9]+)`)
	var s2 = re.ReplaceAllStringFunc(expr, func(s string) string {
		res, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return fmt.Sprintf("%.99f", res)
		}
		return "err"
	})
	return s2
}

func (c *calc) isAssignment(expr string) (bool, string, string) {
	pos1 := strings.Index(expr, "=")
	if pos1 > -1 {
		if pos1 < len(expr)-1 {
			if expr[pos1+1] != '=' {
				ls := strings.TrimSpace(expr[:pos1])
				rs := strings.TrimSpace(expr[pos1+1:])
				if !strings.ContainsAny(ls, " \t()/*-+^") {
					return true, ls, rs
				}
			}
		}
	}
	return false, "", expr
}

func (c *calc) evalExpr(s string) (float64, error) {
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(s, c.functions)
	if err == nil {
		result, err2 := expression.Evaluate(c.parameters)
		if err2 == nil {
			return result.(float64), nil
		}
		return 0, err2
	}
	return 0, err
}

func (c *calc) evalExpression(text1 string) (float64, bool, string, error) {
	text2 := c.replaceexpnumbers(text1)
	text2 = strings.ReplaceAll(text2, "^", "**")
	log.Println("eval repl: ", text2)
	isass, asspara, text3 := c.isAssignment(text2)
	log.Println("eval ass: ", isass, asspara, text3)
	fres, eres := c.evalExpr(text3)
	return fres, isass, asspara, eres
}

const tagres = "   "
const taginf = "?  "
const tagsett = "!"
const tagsettoutformat = tagsett + "outformat"

func (c *calc) f2s(f float64) string {
	return fmt.Sprintf(c.outformat, f)
}

func (c *calc) evaluate() {
	text1 := strings.TrimSpace(c.input.Text)
	log.Println("eval: ", text1)
	if text1 == "help" {
		c.addToHistory(taginf + " Go Calc App https://github.com/wolfgangasdf/gocalcapp")
		c.addToHistory(taginf + " Enter expressions: 1e5*sin(2*pi)")
		c.addToHistory(taginf + " Escape clears input field")
		c.addToHistory(taginf + " Assign variables: a=sqrt(2)")
		c.addToHistory(taginf + " Use variables: sin(2*a)")
		c.addToHistory(taginf + " Show variables: var")
		c.addToHistory(taginf + " Clear history and vars: clr")
		c.addToHistory(taginf + ` Change settings: !outformat=%.8e`)
		c.addToHistory(taginf + " !outformat=" + c.outformat)
		funs := reflect.ValueOf(c.functions).MapKeys() // show functions, not nice
		s := ""
		for i := 0; i < len(funs); i++ {
			if i == len(funs)-1 || (i%5 == 0 && len(s) > 0) {
				c.addToHistory(taginf + "functions: " + s)
				s = ""
			}
			s += funs[i].String() + ", "
		}
		c.input.mySetText("")
		return
	} else if text1 == "clr" {
		c.history.SetText("")
		c.inputHistory = nil
		c.initParameters()
		c.input.mySetText("")
		return
	} else if text1 == "var" {
		c.addToHistory(taginf + "result-variables are not shown!")
		for k, v := range c.parameters {
			if !c.reResVar.MatchString(k) {
				c.addToHistory(taginf + k + " = " + c.f2s(v.(float64)))
			}
		}
		c.input.mySetText("")
		return
	} else if strings.HasPrefix(text1, tagsettoutformat+"=") {
		c.outformat = strings.TrimPrefix(text1, tagsettoutformat+"=")
		c.input.mySetText("")
		return
	}
	// evaluate expression
	c.addToHistory(text1)
	c.inputHistory = append(c.inputHistory, text1)
	fres, isass, asspara, eres := c.evalExpression(text1)
	para := asspara
	if !isass {
		c.lastR++
		para = fmt.Sprintf("r%d", c.lastR)
	}
	if eres == nil {
		c.parameters[para] = fres
		c.addToHistory(tagres + para + "=" + c.f2s(fres))
	} else {
		c.addToHistory(tagres + "error: " + eres.Error())
	}
	c.input.mySetText(c.f2s(fres))
}

func (c *calc) loadUI(app fyne.App) {
	c.input = newInputField(c)
	c.history = newHistoryField(c)
	// history.SetReadOnly(true) // then can't select
	c.scrollhist = container.NewScroll(c.history)
	c.scrollhist.Resize(fyne.NewSize(200, 200))

	c.window = app.NewWindow("Calc")
	// c.window.SetIcon(icon.CalculatorBitmap)
	c.window.Resize(fyne.NewSize(300, 400))

	maincontainer := container.New(layout.NewBorderLayout(
		nil,
		c.input,
		nil, nil),
		c.scrollhist,
		c.input,
	)

	c.loadSettings()

	c.window.SetContent(maincontainer)

	c.window.Show()

	c.window.Canvas().Focus(c.history)
	c.histScrollToEnd()

	c.window.Canvas().Focus(c.input)

	c.window.SetOnClosed(func() {
		c.saveSettings()
	})

}

func (c *calc) loadSettings() {
	log.Println("Load config from ", c.configFile)
	c.outformat = "%g"
	b, err := os.ReadFile(c.configFile)
	if err == nil {
		var htext = ""
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, tagsettoutformat+"=") {
				c.outformat = strings.TrimPrefix(line, tagsettoutformat+"=")
			} else { // reload variable assignments
				htext = htext + line + "\n"
				if strings.HasPrefix(line, tagres) {
					// replace by calling evaluate above? but have to split function...
					isass, asspara, assexpr := c.isAssignment(line)
					if isass {
						f, err := strconv.ParseFloat(assexpr, 64)
						if err == nil {
							c.parameters[asspara] = f
							// update c.lastR
							res := c.reResVar.FindStringSubmatch(asspara)
							if len(res) == 2 {
								res2, err := strconv.Atoi(res[1])
								if err == nil {
									if res2 > c.lastR {
										c.lastR = res2
									}
								}
							}
						} else {
							log.Println("error load line: ", err, line)
						}
					}
				} else if line != "" {
					c.inputHistory = append(c.inputHistory, line)
				}
			}
		}
		c.history.SetText(htext)
	} else {
		log.Println("error reading config: ", err)
	}
}

func (c *calc) saveSettings() {
	log.Println("Save config to ", c.configFile)
	s := ""
	s += tagsettoutformat + "=" + c.outformat + "\n"
	for _, line := range strings.Split(c.history.Text, "\n") {
		if !strings.HasPrefix(line, taginf) && !strings.HasPrefix(line, tagsett) && line != "" {
			s += line + "\n"
		}
	}
	err := os.WriteFile(c.configFile, []byte(s), os.ModePerm)
	if err != nil {
		log.Println("error writing config: ", err)
	}
}

func (c *calc) initParameters() {
	c.lastR = 0
	c.parameters = make(map[string]interface{})
	c.parameters["pi"] = math.Pi
	c.parameters["e"] = math.E
}

func factorial(args ...interface{}) (interface{}, error) {
	xf := args[0].(float64)
	if xf != math.Abs(math.Trunc(xf)) {
		return nil, errors.New("factorial needs positive integer")
	}
	if len(args) != 1 {
		return nil, errors.New("factorial needs one argument")
	}
	x := int(xf)
	f := float64(1)
	for i := 1; i <= x; i++ {
		f *= float64(i)
	}
	return f, nil
}

func newCalculator() *calc {

	c := &calc{}
	c.reResVar = regexp.MustCompile(`r(?P<r>\d+)`)
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("can't load config dir", err)
	}
	os.MkdirAll(configDir+"/gocalcapp", os.ModePerm)
	c.configFile = configDir + "/gocalcapp/gocalcapp.txt"
	fmt.Println("config dir: ", c.configFile)

	c.initParameters()

	c.functions = make(map[string]govaluate.ExpressionFunction)

	checkDoOneArg := func(fun func(float64) float64) func(args ...interface{}) (interface{}, error) {
		return func(args ...interface{}) (interface{}, error) {
			if len(args) != 1 {
				return nil, errors.New("needs one argument")
			}
			return fun(args[0].(float64)), nil
		}
	}

	c.functions["rad"] = checkDoOneArg(func(x float64) float64 { return x / 180.0 * math.Pi })
	c.functions["deg"] = checkDoOneArg(func(x float64) float64 { return 180.0 * x / math.Pi })
	c.functions["sin"] = checkDoOneArg(math.Sin)
	c.functions["cos"] = checkDoOneArg(math.Cos)
	c.functions["tan"] = checkDoOneArg(math.Tan)
	c.functions["asin"] = checkDoOneArg(math.Asin)
	c.functions["acos"] = checkDoOneArg(math.Acos)
	c.functions["atan"] = checkDoOneArg(math.Atan)
	c.functions["sinh"] = checkDoOneArg(math.Sinh)
	c.functions["cosh"] = checkDoOneArg(math.Cosh)
	c.functions["tanh"] = checkDoOneArg(math.Tanh)
	c.functions["asinh"] = checkDoOneArg(math.Asinh)
	c.functions["acosh"] = checkDoOneArg(math.Acosh)
	c.functions["atanh"] = checkDoOneArg(math.Atanh)
	c.functions["ln"] = checkDoOneArg(math.Log)
	c.functions["log2"] = checkDoOneArg(math.Log2)
	c.functions["log"] = checkDoOneArg(math.Log10)
	c.functions["exp"] = checkDoOneArg(math.Exp)
	c.functions["exp2"] = checkDoOneArg(math.Exp2)
	c.functions["sqrt"] = checkDoOneArg(math.Sqrt)
	c.functions["cbrt"] = checkDoOneArg(math.Cbrt)
	c.functions["factorial"] = factorial

	return c
}

// Show app
func Show(app fyne.App) {
	c := newCalculator()
	c.loadUI(app)
}
