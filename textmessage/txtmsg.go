// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package textmessage

import (
	"io"

	"github.com/fractalqb/goxic"
	"golang.org/x/text/message"
)

type Content struct {
	Printer *message.Printer
	Format  string
	Values  []interface{}
}

func (c Content) Emit(wr io.Writer) (n int) {
	n, err := c.Printer.Fprintf(wr, c.Format, c.Values...)
	if err != nil {
		panic(goxic.EmitError{n, err})
	} else {
		return n
	}
}

func Msg(pr *message.Printer, fmt string, values ...interface{}) Content {
	return Content{pr, fmt, values}
}
