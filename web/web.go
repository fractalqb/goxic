// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/goxic"
)

func NewHtmlParser() *goxic.Parser {
	res := goxic.Parser{
		StartInlinePh: "`",
		EndInlinePh:   "`",
		BlockPh: regexp.MustCompile(
			`^[ \t]*<!--(\\?) >>> ([a-zA-Z0-9_-]+) <<< (\\?)-->[ \t]*$`),
		PhNameRgxGrp: 2,
		PhLBrkRgxGrp: 1,
		PhTBrkRgxGrp: 3,
		StartSubTemplate: regexp.MustCompile(
			`^[ \t]*<!--(\\?) >>> ([a-zA-Z0-9_-]+) >>> -->[ \t]*$`),
		StartNameRgxGrp: 2,
		StartLBrkRgxGrp: 1,
		EndSubTemplate: regexp.MustCompile(
			`^[ \t]*<!-- <<< ([a-zA-Z0-9_-]+) <<< (\\?)-->[ \t]*$`),
		EndNameRgxGrp: 1,
		EndTBrkRgxGrp: 2,
		Endl:          "\n"}
	return &res
}

type HtmlEscWriter struct {
	Escape io.Writer
	buf    [utf8.UTFMax]byte
	wp     int
}

func (hew *HtmlEscWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		hew.buf[hew.wp] = b
		hew.wp++
		if buf := hew.buf[:hew.wp]; utf8.FullRune(buf) {
			hew.wp = 0
			if r, _ := utf8.DecodeRune(buf); r == utf8.RuneError {
				return n, errors.New("utf8 rune decoding error")
			} else {
				switch r {
				case '<':
					if i, err := hew.Escape.Write([]byte("&lt;")); err != nil {
						return n + i, err
					} else {
						n += i
					}
				case '>':
					if i, err := hew.Escape.Write([]byte("&gt;")); err != nil {
						return n + i, err
					} else {
						n += i
					}
				case '&':
					if i, err := hew.Escape.Write([]byte("&amp;")); err != nil {
						return n + i, err
					} else {
						n += i
					}
				case '"':
					if i, err := hew.Escape.Write([]byte("&quot;")); err != nil {
						return n + i, err
					} else {
						n += i
					}
				case '\'':
					if i, err := hew.Escape.Write([]byte("&apos;")); err != nil {
						return n + i, err
					} else {
						n += i
					}
				default:
					if i, err := hew.Escape.Write([]byte(string(r))); err != nil {
						return n + i, err
					} else {
						n += i
					}
				}
			}
		}
	}
	return n, nil
}

func EscHtml(str string) string {
	buf := bytes.NewBuffer(nil)
	ewr := HtmlEscWriter{Escape: buf}
	if _, err := ewr.Write([]byte(str)); err != nil {
		panic(err)
	}
	return buf.String()
}

type HtmlEsc struct {
	Cnt goxic.Content
}

func (hc HtmlEsc) Emit(wr io.Writer) int {
	esc := HtmlEscWriter{Escape: wr}
	return hc.Cnt.Emit(&esc)
}

// Span wraps content into a HTML <span></span> element
type Span struct {
	id      string
	class   string
	Wrapped goxic.Content
}

func NewSpan(around goxic.Content, spanId string, spanClass string) *Span {
	res := Span{id: EscHtml(spanId), class: EscHtml(spanClass), Wrapped: around}
	return &res
}

func (s *Span) Emit(wr io.Writer) (n int) {
	if len(s.id) > 0 {
		if len(s.class) > 0 {
			n, err := fmt.Fprintf(wr,
				"<span id=\"%s\" class=\"%s\">",
				s.id,
				s.class)
			if err != nil {
				panic(goxic.EmitError{n, err})
			}
		} else {
			if n, err := fmt.Fprintf(wr, "<span id=\"%s\">", s.id); err != nil {
				panic(goxic.EmitError{n, err})
			}
		}
	} else if len(s.class) > 0 {
		if n, err := fmt.Fprintf(wr, "<span class=\"%s\">", s.class); err != nil {
			panic(goxic.EmitError{n, err})
		}
	} else if n, err := wr.Write([]byte("<span>")); err != nil {
		panic(goxic.EmitError{n, err})
	}
	n += s.Wrapped.Emit(wr)
	if c, err := wr.Write([]byte("</span>")); err != nil {
		panic(goxic.EmitError{n + c, err})
	} else {
		n += c
	}
	return n
}
