// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package web

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/fractalqb/goxic"
	"github.com/stvp/assert"
)

func TestHtmlEscWriter_Write(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ewr := HtmlEscWriter{Escape: buf}
	n, err := ewr.Write([]byte("<>&\"'"))
	assert.Nil(t, err, "have error: ", err)
	assert.Equal(t, 25, n, "expected bytes written")
	assert.Equal(t, "&lt;&gt;&amp;&quot;&apos;", buf.String(), "wrong output")
}

func TestHtmlEscWriter_umls(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ewr := HtmlEscWriter{Escape: buf}
	n, err := ewr.Write([]byte("öäüß"))
	assert.Nil(t, err, "have error: ", err)
	assert.Equal(t, 8, n, "expected bytes written")
	assert.Equal(t, "öäüß", buf.String(), "wrong output")
}

func BenchmarkHtmlEscWriter_umls(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	ewr := HtmlEscWriter{Escape: buf}
	txt := []byte("öäüß")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ewr.Write(txt)
	}
}

func ExampleEscape() {
	tmpl := goxic.NewTemplate("")
	tmpl.Placeholder("html")
	bt := tmpl.NewBounT(nil)
	bt.BindName("html", HtmlEsc{goxic.Print{"<&\"'>"}})
	bt.Emit(os.Stdout)
	// Output:
	// &lt;&amp;&quot;&apos;&gt;
}

func ExampleSpan() {
	out := os.Stdout
	cnt := goxic.Print{"foo"}
	span := NewSpan(cnt, "", "")
	span.Emit(out)
	fmt.Fprintln(out)
	span = NewSpan(cnt, "&ID", "")
	span.Emit(out)
	fmt.Fprintln(out)
	span = NewSpan(cnt, "", "'CLS")
	span.Emit(out)
	fmt.Fprintln(out)
	span = NewSpan(cnt, "&ID", "'CLS")
	span.Emit(out)
	fmt.Fprintln(out)
	// Output:
	// <span>foo</span>
	// <span id="&amp;ID">foo</span>
	// <span class="&apos;CLS">foo</span>
	// <span id="&amp;ID" class="&apos;CLS">foo</span>
}
