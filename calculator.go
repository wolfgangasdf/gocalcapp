package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"github.com/Knetic/govaluate"
)

func NewInputField(c *calc) *InputField {
	search := widget.NewEntry()
	search.SetPlaceHolder(`Expression or "help"`)
	return &InputField{search, c}
}

type InputField struct {
	*widget.Entry
	c *calc
}

func (s *InputField) walkHistory(diff int) {
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

func (s *InputField) TypedKey(key *fyne.KeyEvent) {
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

func (s *InputField) CreateRenderer() fyne.WidgetRenderer {
	return widget.Renderer(s.Entry)
}

func (s *InputField) mySetText(t string) {
	s.Entry.SetText(t)
	s.c.window.Canvas().Refresh(s.c.window.Content()) // important, bug?
	if t != "" {                                      // select all
		s.Entry.TypedKey(&fyne.KeyEvent{fyne.KeyHome})
		s.Entry.KeyDown(&fyne.KeyEvent{desktop.KeyShiftLeft})
		s.Entry.TypedKey(&fyne.KeyEvent{fyne.KeyEnd})
		s.Entry.KeyUp(&fyne.KeyEvent{desktop.KeyShiftLeft})
	}
}

type calc struct {
	input        *InputField
	inputHistory []string
	inputHistPos int
	history      *widget.Entry
	scrollhist   *widget.ScrollContainer
	window       fyne.Window
	functions    map[string]govaluate.ExpressionFunction
	parameters   map[string]interface{} // variables
	configFile   string
	lastR        int // highest used rXXX variable
	reResVar     *regexp.Regexp
}

func (c *calc) histScrollToEnd() {
	for range "12" { // bug? needs this twice or not scrolled far enough
		c.scrollhist.Scrolled(&fyne.ScrollEvent{fyne.PointEvent{}, 0, -c.history.Size().Height})
	}
	c.window.Canvas().Refresh(c.window.Content()) // important, bug?
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
		} else {
			return "err"
		}
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
		} else {
			return 0, err2
		}
	} else {
		return 0, err
	}
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

func (c *calc) f2s(f float64) string {
	return fmt.Sprintf("%.14g", f)
}

func (c *calc) evaluate() {
	text1 := strings.TrimSpace(c.input.Text)
	log.Println("eval: ", text1)
	if text1 == "help" {
		c.addToHistory(taginf + " Go Calc App")
		c.addToHistory(taginf + " Enter expressions: 1e5*sin(2*pi)")
		c.addToHistory(taginf + " Escape clears input field")
		c.addToHistory(taginf + " Assign variables: a=sqrt(2)")
		c.addToHistory(taginf + " Use variables: sin(2*a)")
		c.addToHistory(taginf + " Show variables: var")
		c.addToHistory(taginf + " Clear history and vars: clr")
		funs := reflect.ValueOf(c.functions).MapKeys() // not nice
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
	}
	// evaluate expression
	c.addToHistory(text1)
	c.inputHistory = append(c.inputHistory, text1)
	fres, isass, asspara, eres := c.evalExpression(text1)
	para := asspara
	if !isass {
		c.lastR += 1
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

func (c *calc) addButton(text string, action func()) *widget.Button {
	button := widget.NewButton(text, action)

	return button
}

func (c *calc) loadUI(app fyne.App) {
	c.input = NewInputField(c)
	c.history = widget.NewMultiLineEntry()
	// history.SetReadOnly(true) // then can't select
	c.scrollhist = widget.NewScrollContainer(c.history)
	c.scrollhist.Resize(fyne.NewSize(200, 200))

	c.window = app.NewWindow("Calc")
	// c.window.SetIcon(icon.CalculatorBitmap)
	c.window.Resize(fyne.NewSize(300, 400))

	maincontainer := fyne.NewContainerWithLayout(layout.NewBorderLayout(
		nil,
		c.input,
		nil, nil),
		c.scrollhist,
		c.input,
	)

	c.loadSettings()

	c.window.SetContent(maincontainer)

	c.window.Show()

	c.histScrollToEnd()

	c.window.Canvas().Focus(c.input)

	c.window.SetOnClosed(func() {
		c.saveSettings()
	})

}

func (c *calc) loadSettings() {
	log.Println("Load config from ", c.configFile)

	b, err := ioutil.ReadFile(c.configFile)
	if err == nil {
		c.history.SetText(string(b))
		for _, line := range strings.Split(c.history.Text, "\n") {
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
	} else {
		log.Println("error reading config: ", err)
	}
}

func (c *calc) saveSettings() {
	log.Println("Save config to ", c.configFile)
	s := ""
	for _, line := range strings.Split(c.history.Text, "\n") {
		if !strings.HasPrefix(line, taginf) && line != "" {
			s += line + "\n"
		}
	}
	err := ioutil.WriteFile(c.configFile, []byte(s), os.ModePerm)
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
				return nil, errors.New("sin needs one argument, have ")
			} else {
				return fun(args[0].(float64)), nil
			}
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

	return c
}

func Show(app fyne.App) {
	c := newCalculator()
	c.loadUI(app)
}
