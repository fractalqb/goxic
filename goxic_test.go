// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package goxic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/stvp/assert"
)

func assertIndices(t *testing.T, got []int, expect ...int) {
	if got == nil {
		t.Errorf("no indices for placeholder")
	} else if len(got) != len(expect) {
		t.Errorf("expected %d indices for placeholder, got %d",
			len(expect),
			len(got))
	} else {
		for p, idx := range expect {
			if idx != got[p] {
				t.Errorf("wrong index: expected %d, got %d", idx, got[p])
			}
		}
	}
}

func mustStr(str string) string {
	if len(str) > 0 {
		return str
	} else {
		return "<error>"
	}
}

func TestEmptyTemplate(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	assert.Equal(t, 0, tmpl.FixCount())
	assert.Equal(t, 0, len(tmpl.Placeholders()))
}

func TestOneFix(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("Fixate")
	assert.Equal(t, 1, tmpl.FixCount())
	assert.Equal(t, 0, len(tmpl.Placeholders()))
	assert.Equal(t, "Fixate", string(tmpl.FixAt(0)))
}

func TestMergeFix(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("<thisisfix1>")
	tmpl.AddStr("<thisisfix2>")
	assert.Equal(t, 1, tmpl.FixCount())
	assert.Equal(t, "<thisisfix1><thisisfix2>", string(tmpl.FixAt(0)))
}

func TestLeadingPlaceholder(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.Placeholder("foo")
	tmpl.AddFix(fragment("bar"))
	assert.Equal(t, "foo", mustStr(tmpl.PlaceholderAt(0)), "placeholder")
	assertIndices(t, tmpl.PlaceholderIdxs("foo"), 0)
	assert.Equal(t, "bar", string(tmpl.FixAt(0)), "fix")
}

func TestTrailingPlaceholder(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddFix(fragment("foo"))
	tmpl.Placeholder("bar")
	assert.Equal(t, "bar", mustStr(tmpl.PlaceholderAt(1)), "placeholder")
	assertIndices(t, tmpl.PlaceholderIdxs("bar"), 1)
	assert.Equal(t, "foo", string(tmpl.FixAt(0)), "fix")
}

func TestMidPlaceholder(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddFix(fragment("foo"))
	tmpl.Placeholder("bar")
	tmpl.AddFix(fragment("baz"))
	assert.Equal(t, "foo", string(tmpl.FixAt(0)), "fix")
	assert.Equal(t, "baz", string(tmpl.FixAt(1)), "fix")
	assertIndices(t, tmpl.PlaceholderIdxs("bar"), 1)
}

func TestTwoPlaceholders(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.Placeholder("foo")
	tmpl.Placeholder("bar")
	assert.Equal(t, 1, tmpl.FixCount(), "fixed fragments")
	assertIndices(t, tmpl.PlaceholderIdxs("foo"), 0)
	assertIndices(t, tmpl.PlaceholderIdxs("bar"), 1)
}

func TestCatchEmit(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("begin\n")
	tmpl.Placeholder("foo")
	tmpl.AddStr("end")
	bt := tmpl.NewBounT(nil)
	bt.BindGenName("foo", func(wr io.Writer) int {
		panic(EmitError{4711, errors.New("fails")})
	})
	n, err := CatchEmit(bt, os.Stdout)
	assert.NotNil(t, err)
	assert.Equal(t, 4711, n)
	assert.Equal(t, "fails", err.Error())
}

func TestAnonymousBindFails(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("foo")
	bt := tmpl.NewBounT(nil)
	err := bt.BindP([]int{0}, 4711)
	assert.NotNil(t, err)
}

func ExampleBounT() {
	tmpl := NewTemplate("")
	tmpl.Placeholder("foo")
	tmpl.AddStr("<thisisfix1>")
	tmpl.Placeholder("foo")
	tmpl.AddStr("<thisisfix2>")
	tmpl.Placeholder("bar")
	bnt := tmpl.NewBounT(nil)
	bnt.BindPName("foo", "FOO")
	bnt.BindPName("bar", "BAR")
	bnt.Emit(os.Stdout)
	// Output:
	// FOO<thisisfix1>FOO<thisisfix2>BAR
}

func ExampleDynamicContent() {
	ts := "2017-11-11 19:18:49"
	tmpl := NewTemplate("")
	tmpl.AddStr("It's now ")
	tmpl.Placeholder("timestamp")
	bt := tmpl.NewBounT(nil)
	bt.BindName("timestamp", Generator(func(wr io.Writer) int {
		if n, err := fmt.Fprint(wr, ts); err != nil {
			panic(EmitError{n, err})
		} else {
			return n
		}
	}))
	bt.Emit(os.Stdout)
	// Output:
	// It's now 2017-11-11 19:18:49
}

func ExampleFixate() {
	tr := NewTemplate("root")
	tr.AddStr("foo")
	tr.Placeholder("bar")
	tr.AddStr("baz")
	tn := NewTemplate("sub")
	tn.AddStr("N-TMPL")
	tn.Placeholder("quux")
	bt := tr.NewBounT(nil)
	bt.BindName("bar", tn.NewBounT(nil))
	ft := bt.Fixate()
	for i, ph := range ft.Placeholders() {
		fmt.Printf("%d: [%s]\n", i, ph)
	}
	bt = ft.NewInitBounT(Print{"___"}, nil)
	bt.Emit(os.Stdout)
	// Output:
	// 0: [sub:quux]
	// fooN-TMPL___baz
}

type IMap struct {
	*Template
	Foo []int // use naming convention to map, same as `goxic:"Foo"`
	Bar []int `goxic:"bar"`
	Baz []int `goxic:"baz opt"`
}

func TestIndexMap(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.Placeholder("Foo")
	tmpl.Placeholder("bar")
	tmpl.Placeholder("quux")
	var imap IMap
	unmappend := InitIndexMap(&imap, tmpl, IdName)
	assert.Equal(t, tmpl, imap.Template)
	assertIndices(t, imap.Foo, 0)
	assertIndices(t, imap.Bar, 1)
	assertIndices(t, imap.Baz)
	assert.Equal(t, 1, len(unmappend.Placeholders))
	assert.Equal(t, "quux", unmappend.Placeholders[0])
}

func TestTemplate_RenamePh(t *testing.T) {
	tmpl := NewTemplate("rename")
	tmpl.AddStr("AB")
	tmpl.Placeholder("a")
	tmpl.AddStr("CD")
	tmpl.Placeholder("b")
	tmpl.AddStr("EF")
	tmpl.Placeholder("a")
	tmpl.AddStr("GH")
	tmpl.Placeholder("b")
	tmpl.AddStr("IJ")
	err := tmpl.RenamePh("unknown", "foo", true)
	if err == nil || err.Error() != "template has no placeholder 'unknown'" {
		t.Fatalf("expected unkonwn placeholder, got: %v", err)
	}
	err = tmpl.RenamePh("a", "b", false)
	if err == nil || err.Error() != "cannot rename 'a', new name 'b' already exists" {
		t.Fatalf("expected already exists, got: %v", err)
	}
	err = tmpl.RenamePh("a", "c", false)
	if err != nil {
		t.Fatalf("unexpected expected error a → c: %v", err)
	}
	err = tmpl.RenamePh("b", "c", true)
	if err != nil {
		t.Fatalf("unexpected expected error b → c: %v", err)
	}
	if !reflect.DeepEqual(tmpl.Placeholders(), []string{"c"}) {
		t.Fatalf("wrong placeholders: %v", tmpl.Placeholders())
	}
}

// Would be nice to work…
//type GxtNest struct {
//	*Template
//	Id []int
//}

//type gxtFinal struct {
//	GxtNest
//	Name []int
//}

//func TestIndexMap_nest(t *testing.T) {
//	tmpl := NewTemplate(t.Name())
//	tmpl.Placeholder("Id")
//	tmpl.Placeholder("Name")
//	var imap gxtFinal
//	unmapped := InitIndexMap(&imap, tmpl, IdName)
//	assert.Equal(t, 0, len(unmapped.Placeholders), unmapped.Placeholders)
//}
