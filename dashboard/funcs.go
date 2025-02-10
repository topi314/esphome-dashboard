package dashboard

import (
	"fmt"
	"html/template"
	"strings"
)

func seq(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func convertNewLinesToBR(a any) string {
	return strings.ReplaceAll(fmt.Sprint(a), "\n", "<br>")
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func safeCSS(s string) template.CSS {
	return template.CSS(s)
}

func safeHTMLAttr(s string) template.HTMLAttr {
	return template.HTMLAttr(s)
}

func safeURL(s string) template.URL {
	return template.URL(s)
}

func safeJS(s string) template.JS {
	return template.JS(s)
}

func safeJSStr(s string) template.JSStr {
	return template.JSStr(s)
}

func safeSrcset(s string) template.Srcset {
	return template.Srcset(s)
}
