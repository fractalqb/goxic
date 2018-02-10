// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package goxic

import (
	"fmt"
	"os"
	"strings"

	"regexp"
	"testing"

	"github.com/stvp/assert"
)

func newTestParser() *Parser {
	res := Parser{
		StartInlinePh: "`",
		EndInlinePh:   "`",
		BlockPh: regexp.MustCompile(
			"^[ \\t]*<!--(\\\\?) >>> ([a-zA-Z0-9_-]+) <<< (\\\\?)-->[ \\t]*$"),
		PhNameRgxGrp: 2,
		PhLBrkRgxGrp: 1,
		PhTBrkRgxGrp: 3,
		StartSubTemplate: regexp.MustCompile(
			"^[ \\t]*<!--(\\\\?) >>> ([a-zA-Z0-9_-]+) >>> -->[ \\t]*$"),
		StartNameRgxGrp: 2,
		StartLBrkRgxGrp: 1,
		EndSubTemplate: regexp.MustCompile(
			"^[ \\t]*<!-- <<< ([a-zA-Z0-9_-]+) <<< (\\\\?)-->[ \\t]*$"),
		EndNameRgxGrp: 1,
		EndTBrkRgxGrp: 2,
		Endl:          "\n"}
	return &res
}

func assertEndls(t *testing.T,
	fix []byte, expectLeading, expectTrailing bool,
	hints ...interface{}) {
	txt := string(fix)
	assert.Equal(t,
		expectLeading,
		strings.HasPrefix(txt, "\n"),
		"leading linebreak")
	assert.Equal(t,
		expectTrailing,
		strings.HasSuffix(txt, "\n"),
		"leading linebreak")

}

func ExampleRegexp() {
	rgx := regexp.MustCompile("^[ \\t]*<!--(\\\\?) >>> ([a-zA-Z_-]+) <<< -->[ \\t]*$")
	match := rgx.FindStringSubmatch("<!--\\ >>> x <<< -->")
	for i, m := range match {
		fmt.Printf("%d: [%s]\n", i, m)
	}
	match = rgx.FindStringSubmatch("<!-- >>> x <<< -->")
	for i, m := range match {
		fmt.Printf("%d: [%s]\n", i, m)
	}
	// Output:
	// 0: [<!--\ >>> x <<< -->]
	// 1: [\]
	// 2: [x]
	// 0: [<!-- >>> x <<< -->]
	// 1: []
	// 2: [x]
}

func TestPathStr_nil(t *testing.T) {
	var path []string = nil
	pstr := pathStr(path)
	assert.Equal(t, "", pstr)
}

func TestPathStr_empty(t *testing.T) {
	path := []string{}
	pstr := pathStr(path)
	assert.Equal(t, "", pstr)
}

func TestPathStr_single(t *testing.T) {
	path := []string{"foo"}
	pstr := pathStr(path)
	assert.Equal(t, "foo", pstr)
}

func TestPathStr_two(t *testing.T) {
	path := []string{"foo", "bar"}
	pstr := pathStr(path)
	assert.Equal(t, "foo/bar", pstr)
}

func TestParser_addLine(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	p := newTestParser()
	p.addLine(tmpl, "foo`bar`baz")
	assert.Equal(t, 2, tmpl.FixCount())
	assert.Equal(t, "foo", string(tmpl.FixAt(0)))
	assert.Equal(t, "baz", string(tmpl.FixAt(1)))
	assertIndices(t, tmpl.PlaceholderIdxs("bar"), 1)
}

func ExampleParser() {
	rd := strings.NewReader(`1st line of TL template
<!-- >>> subt1 >>> -->
1st line of subt1
<!-- >>> sub1ph <<< -->
<!-- <<< subt1 <<< \-->
last line of TL template

`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, "", ts); err != nil {
		fmt.Printf("parsing failed: %s\n", err)
	} else {
		tlt := ts[""]
		if _, err := CatchEmit(tlt.NewBounT(), os.Stdout); err != nil {
			fmt.Println(err)
		}
		sbt := ts["subt1"]
		sbbt := sbt.NewBounT()
		sbbt.BindPName("sub1ph", "SUB1PH")
		if _, err := CatchEmit(sbbt, os.Stdout); err != nil {
			fmt.Println(err)
		}
	}
	// Output:
	// 1st line of TL template
	// last line of TL template
	// 1st line of subt1
	// SUB1PH
}

func Test1LineNoBrk(t *testing.T) {
	rd := strings.NewReader("line1")
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assert.Equal(t, 1, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, false)
	}
}

func Test1LineLBrk(t *testing.T) {
	rd := strings.NewReader("\nline1")
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assert.Equal(t, 1, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), true, false)
	}
}

func Test1LineTBrk(t *testing.T) {
	rd := strings.NewReader("line1\n\n")
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assert.Equal(t, 1, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, true)
	}
}

func Test1LineLnTBrk(t *testing.T) {
	rd := strings.NewReader("\nline1\n\n")
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assert.Equal(t, 1, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), true, true)
	}
}

func TestInlinePlaceholderNoBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!--\ >>> phnm <<< \-->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assertIndices(t, tmpl.PlaceholderIdxs("phnm"), 1)
		assert.Equal(t, 2, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, false)
		assertEndls(t, tmpl.FixAt(1), false, false)
	}
}

func TestInlinePlaceholderLBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!-- >>> phnm <<< \-->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assertIndices(t, tmpl.PlaceholderIdxs("phnm"), 1)
		assert.Equal(t, 2, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, true)
		assertEndls(t, tmpl.FixAt(1), false, false)
	}
}

func TestInlinePlaceholderTBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!--\ >>> phnm <<< -->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assertIndices(t, tmpl.PlaceholderIdxs("phnm"), 1)
		assert.Equal(t, 2, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, false)
		assertEndls(t, tmpl.FixAt(1), true, false)
	}
}

func TestInlinePlaceholderLnTBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!-- >>> phnm <<< -->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tmpl := ts[""]
		assertIndices(t, tmpl.PlaceholderIdxs("phnm"), 1)
		assert.Equal(t, 2, tmpl.FixCount())
		assertEndls(t, tmpl.FixAt(0), false, true)
		assertEndls(t, tmpl.FixAt(1), true, false)
	}
}

func TestNestmplNoBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!--\ >>> sub >>> -->
subln
<!-- <<< sub <<< \-->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "line1line2", string(rtmpl.FixAt(0)))
	}
}

func TestNestmplLBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!-- >>> sub >>> -->
subln
<!-- <<< sub <<< \-->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "line1\nline2", string(rtmpl.FixAt(0)))
	}
}

func TestNestmplTBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!--\ >>> sub >>> -->
subln
<!-- <<< sub <<< -->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "line1\nline2", string(rtmpl.FixAt(0)))
	}
}

func TestNestmplLnTBrk(t *testing.T) {
	rd := strings.NewReader(`line1
<!-- >>> sub >>> -->
subln
<!-- <<< sub <<< -->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "line1\n\nline2", string(rtmpl.FixAt(0)))
	}
}

func TestNestmplAsStart(t *testing.T) {
	rd := strings.NewReader(`<!-- >>> sub >>> -->
subln
<!-- <<< sub <<< -->
line2`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "\nline2", string(rtmpl.FixAt(0)))
	}
}

func TestNestmplAsEnd(t *testing.T) {
	rd := strings.NewReader(`line1
<!-- >>> sub >>> -->
subln
<!-- <<< sub <<< -->`)
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		nestp := ts["sub"]
		assert.Equal(t, 1, nestp.FixCount())
		assertEndls(t, nestp.FixAt(0), false, false)
		rtmpl := ts[""]
		assert.Equal(t, 1, rtmpl.FixCount())
		assert.Equal(t, "line1\n", string(rtmpl.FixAt(0)))
	}
}

func TestOnlyPlaceholder(t *testing.T) {
	rd := strings.NewReader("`foo`")
	p := newTestParser()
	ts := make(map[string]*Template)
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		tp := ts[""]
		assert.NotNil(t, tp)
		assert.Equal(t, 0, tp.FixCount())
		assert.Equal(t, 1, tp.PlaceholderNum())
		assert.Equal(t, "foo", tp.PlaceholderAt(0))
	}
}

func TestEmptyNestedTemplate(t *testing.T) {
	rd := strings.NewReader(`foo
<!-- >>> empty >>> -->
<!-- <<< empty <<< -->
bar`)
	ts := make(map[string]*Template)
	p := newTestParser()
	if err := p.Parse(rd, t.Name(), ts); err != nil {
		t.Fatalf("cannot parse template: %s", err)
	} else {
		assert.Equal(t, 1, len(ts))
	}
}
