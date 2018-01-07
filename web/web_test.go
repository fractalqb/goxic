package web

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"git.fractalqb.de/goxic"
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

func ExampleEscape() {
	tmpl := goxic.NewTemplate("")
	tmpl.Placeholder("html")
	bt := tmpl.NewBounT()
	bt.BindName("html", EscHtml{goxic.Print{"<&\"'>"}})
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
