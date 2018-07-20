// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017-2018 Marcus Perlick
package html

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"git.fractalqb.de/fractalqb/goxic"
	"github.com/stvp/assert"
)

func TestHtmlEscWriter_Write(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ewr := EscWriter{Escape: buf}
	n, err := ewr.Write([]byte("<>&\"'"))
	assert.Nil(t, err, "have error: ", err)
	assert.Equal(t, 25, n, "expected bytes written")
	assert.Equal(t, "&lt;&gt;&amp;&quot;&apos;", buf.String(), "wrong output")
}

func TestHtmlEscWriter_umls(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	ewr := EscWriter{Escape: buf}
	n, err := ewr.Write([]byte("öäüß"))
	assert.Nil(t, err, "have error: ", err)
	assert.Equal(t, 8, n, "expected bytes written")
	assert.Equal(t, "öäüß", buf.String(), "wrong output")
}

func BenchmarkHtmlEscWriter_umls(b *testing.B) {
	buf := bytes.NewBuffer(nil)
	ewr := EscWriter{Escape: buf}
	txt := []byte("öäüß")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ewr.Write(txt)
	}
}

func ExampleEscape() {
	tmpl := goxic.NewTemplate("")
	tmpl.Ph("html")
	bt := tmpl.NewBounT(nil)
	bt.BindName("html", Escaper{goxic.Print{"<&\"'>"}})
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
