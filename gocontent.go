// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017-2018 Marcus Perlick
package goxic

import (
	"fmt"
	"io"
)

type Generator func(wr io.Writer) int

func (f Generator) Emit(wr io.Writer) int {
	return f(wr)
}

func (bt *BounT) BindGen(phIdxs []int, f func(wr io.Writer) int) int {
	return bt.Bind(phIdxs, Generator(f))
}

func (bt *BounT) BindGenName(name string,
	f func(wr io.Writer) int) error {
	return bt.BindName(name, Generator(f))
}

func (bt *BounT) BindGenIfName(name string,
	f func(wr io.Writer) int) {
	bt.BindIfName(name, Generator(f))
}

type fmtCnt struct {
	fmt string
	val []interface{}
}

func Printf(fmt string, vs ...interface{}) Content {
	return fmtCnt{fmt, vs}
}

func (fc fmtCnt) Emit(wr io.Writer) int {
	if n, err := fmt.Fprintf(wr, fc.fmt, fc.val...); err != nil {
		panic(EmitError{n, err})
	} else {
		return n
	}
}

func (bt *BounT) BindFmt(phIdxs []int, fmt string, vals ...interface{}) int {
	return bt.Bind(phIdxs, fmtCnt{fmt, vals})
}

func (bt *BounT) BindFmtName(name string, fmt string, vals ...interface{}) error {
	return bt.BindName(name, fmtCnt{fmt, vals})
}

func (bt *BounT) BindFmtIfName(name string, fmt string, vals ...interface{}) {
	bt.BindIfName(name, fmtCnt{fmt, vals})
}

type Print struct {
	V interface{}
}

func (c Print) Emit(wr io.Writer) int {
	if n, err := fmt.Fprint(wr, c.V); err != nil {
		panic(EmitError{n, err})
	} else {
		return n
	}
}

func (bt *BounT) BindP(phIdxs []int, printable interface{}) int {
	return bt.Bind(phIdxs, Print{printable})
}

func (bt *BounT) BindPName(name string, printable interface{}) error {
	return bt.BindName(name, Print{printable})
}

func (bt *BounT) BindPIfName(name string, printable interface{}) {
	bt.BindIfName(name, Print{printable})
}

type Data []byte

func (d Data) Emit(wr io.Writer) int {
	n, err := wr.Write(d)
	if err != nil {
		panic(EmitError{n, err})
	} else {
		return n
	}
}

type Embracer struct {
	Prefix  []byte
	Cnt     Content
	Postfix []byte
}

func (e *Embracer) Emit(wr io.Writer) (res int) {
	if e.Prefix != nil {
		n, err := wr.Write(e.Prefix)
		if err != nil {
			panic(EmitError{n, err})
		}
		res = n
	}
	res += e.Cnt.Emit(wr)
	if e.Postfix != nil {
		n, err := wr.Write(e.Postfix)
		if err != nil {
			panic(EmitError{n, err})
		}
		res += n
	}
	return res
}

func Embrace(prefix string, c Content, postfix string) Embracer {
	return Embracer{
		Prefix:  []byte(prefix),
		Cnt:     c,
		Postfix: []byte(postfix),
	}
}

func (e *Embracer) Wrap(c Content) Content {
	return &Embracer{
		Prefix:  e.Prefix,
		Cnt:     c,
		Postfix: e.Postfix,
	}
}
