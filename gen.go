// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func genasset(file string, outfile string, funcname string) {
	data, err := ioutil.ReadFile(file)
	fmt.Println("Generating " + outfile + " from " + file + "...")
	if err != nil {
		panic(err)
	}
	s := "package main\n// Code generated DO NOT EDIT.\nfunc " + funcname + "() []byte {\nreturn []byte{ "
	for _, b := range data {
		s += fmt.Sprintf("0x%.2x, ", b)
	}
	s += "}\n}"
	ioutil.WriteFile(outfile, []byte(s), os.ModePerm)
}

func main() {
	genasset("icon.png", "icon-gen.go", "icon")
}
