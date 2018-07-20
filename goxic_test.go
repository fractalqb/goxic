// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017-2018 Marcus Perlick
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
	assert.Equal(t, 0, len(tmpl.Phs()))
}

func TestOneFix(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("Fixate")
	assert.Equal(t, 1, tmpl.FixCount())
	assert.Equal(t, 0, len(tmpl.Phs()))
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
	tmpl.Ph("foo")
	tmpl.AddFix(fragment("bar"))
	assert.Equal(t, "foo", mustStr(tmpl.PhAt(0)), "placeholder")
	assertIndices(t, tmpl.PhIdxs("foo"), 0)
	assert.Equal(t, "bar", string(tmpl.FixAt(0)), "fix")
}

func TestTrailingPlaceholder(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddFix(fragment("foo"))
	tmpl.Ph("bar")
	assert.Equal(t, "bar", mustStr(tmpl.PhAt(1)), "placeholder")
	assertIndices(t, tmpl.PhIdxs("bar"), 1)
	assert.Equal(t, "foo", string(tmpl.FixAt(0)), "fix")
}

func TestMidPlaceholder(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddFix(fragment("foo"))
	tmpl.Ph("bar")
	tmpl.AddFix(fragment("baz"))
	assert.Equal(t, "foo", string(tmpl.FixAt(0)), "fix")
	assert.Equal(t, "baz", string(tmpl.FixAt(1)), "fix")
	assertIndices(t, tmpl.PhIdxs("bar"), 1)
}

func TestTwoPlaceholders(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.Ph("foo")
	tmpl.Ph("bar")
	assert.Equal(t, 1, tmpl.FixCount(), "fixed fragments")
	assertIndices(t, tmpl.PhIdxs("foo"), 0)
	assertIndices(t, tmpl.PhIdxs("bar"), 1)
}

func TestCatchEmit(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.AddStr("begin\n")
	tmpl.Ph("foo")
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
	tmpl := NewTemplate("").
		Ph("foo").
		AddStr("<thisisfix1>").
		Ph("foo").
		AddStr("<thisisfix2>").
		Ph("bar")
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
	tmpl.AddStr("It's now ").Ph("timestamp")
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
	tr := NewTemplate("root").AddStr("foo").Ph("bar").AddStr("baz")
	tn := NewTemplate("sub").AddStr("N-TMPL").Ph("quux")
	bt := tr.NewBounT(nil)
	bt.BindName("bar", tn.NewBounT(nil))
	ft := bt.Fixate()
	for i, ph := range ft.Phs() {
		fmt.Printf("%d: [%s]\n", i, ph)
	}
	bt = ft.NewInitBounT(Print{"___"}, nil)
	bt.Emit(os.Stdout)
	// Output:
	// 0: [sub:quux]
	// fooN-TMPL___baz
}

func TestTemplate_RenamePh(t *testing.T) {
	tmpl := NewTemplate("rename")
	tmpl.AddStr("AB")
	tmpl.Ph("a")
	tmpl.AddStr("CD")
	tmpl.Ph("b")
	tmpl.AddStr("EF")
	tmpl.Ph("a")
	tmpl.AddStr("GH")
	tmpl.Ph("b")
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
	if !reflect.DeepEqual(tmpl.Phs(), []string{"c"}) {
		t.Fatalf("wrong placeholders: %v", tmpl.Phs())
	}
}

func ExampleTemplate_XformPhs() {
	tmpl := NewTemplate("rename")
	tmpl.AddStr("AB")
	tmpl.Ph("a")
	tmpl.AddStr("CD")
	tmpl.Ph("b")
	tmpl.AddStr("EF")
	tmpl.Ph("c")
	tmpl.AddStr("GH")
	tmpl.Ph("d")
	tmpl.AddStr("IJ")
	tmpl.XformPhs(true, func(pi string) string {
		switch pi {
		case "a":
			return "d"
		case "d":
			return "a"
		default:
			return "b"
		}
	})
	bt := tmpl.NewBounT(nil)
	bt.BindPName("a", "<A>")
	bt.BindPName("b", "<B>")
	bt.BindPName("d", "<D>")
	bt.Emit(os.Stdout)
	// Output:
	// AB<D>CD<B>EF<B>GH<A>IJ
}

func ExampleTemplate_Wrap() {
	tmpl := NewTemplate("embrace").AddStr("foo").Ph("bar").AddStr("baz")
	e := Embrace("<[", nil, "]>")
	tmpl.Wrap(e.Wrap, tmpl.PhIdxs("bar")...)
	bt := tmpl.NewInitBounT(Print{"BAR"}, nil)
	bt.Emit(os.Stdout)
	// Output:
	// foo<[BAR]>baz
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
