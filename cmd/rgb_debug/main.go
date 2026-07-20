package main

import (
	"fmt"
	"wb-ui/css"
)

func main() {
	tests := []string{
		"rgb(0,0,255)",
		"rgb(255,0,0)",
		"hsl(120,100%,50%)",
		"rgba(255,0,0,0.5)",
		"#ff0000",
		"background:rgb(0,0,255)",
	}
	for _, s := range tests {
		p := css.NewParser(s)
		decls := p.ParseDeclarationList()
		for _, d := range decls {
			vs := d.ValueString()
			fmt.Printf("input: %q\n  tokens: ", s)
			for _, tok := range d.Value {
				fmt.Printf("%+v ", tok)
			}
			fmt.Printf("\n  ValueString: %q\n\n", vs)
		}
	}
}
