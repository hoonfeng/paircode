// 验证 wb-ui HTML 解析器对属性值含 && 的处理
//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"wb-ui/dom"
	"wb-ui/html"
)

func dumpNode(n dom.Node, indent string) {
	switch v := n.(type) {
	case *dom.Element:
		attrs := []string{}
		for _, name := range v.AttributeNames() {
			attrs = append(attrs, name+"="+v.GetAttribute(name))
		}
		fmt.Printf("%s<%s %s> children=%d\n", indent, v.LocalName(), strings.Join(attrs, " "), len(v.ChildNodes()))
		for _, c := range v.ChildNodes() {
			dumpNode(c, indent+"  ")
		}
	case *dom.Text:
		fmt.Printf("%sTEXT: %q\n", indent, v.Data())
	case *dom.Comment:
		fmt.Printf("%sCOMMENT: %q\n", indent, v.Data())
	case *dom.Document:
		fmt.Printf("%sDOCUMENT\n", indent)
		for _, c := range v.ChildNodes() {
			dumpNode(c, indent+"  ")
		}
	default:
		fmt.Printf("%s<other %T>\n", indent, n)
	}
}

func main() {
	cases := []string{
		`<div><span v-if="a && b" class="x">{{ t }}</span></div>`,
		`<div><span v-if="a & b" class="x">t</span></div>`,
		`<div><span v-if="a&amp;b" class="x">t</span></div>`,
		`<div><template v-if="a && b"><span>x</span></template></div>`,
		`<div><span class="a" v-if="x">ok</span></div>`,
	}
	for i, src := range cases {
		fmt.Printf("========== case %d: %s\n", i, src)
		doc, err := html.Parse(src)
		if err != nil {
			fmt.Printf("  Parse err: %v\n", err)
			continue
		}
		fmt.Printf("  innerHTML: %q\n", doc.DocumentElement().GetInnerHTML())
		fmt.Printf("  DOM tree:\n")
		dumpNode(doc, "  ")
	}
	_ = os.Stdout
}
