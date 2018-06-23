# goxic
[![Build Status](https://travis-ci.org/fractalqb/goxic.svg)](https://travis-ci.org/fractalqb/goxic)
[![codecov](https://codecov.io/gh/fractalqb/goxic/branch/master/graph/badge.svg)](https://codecov.io/gh/fractalqb/goxic)
[![Go Report Card](https://goreportcard.com/badge/github.com/fractalqb/goxic)](https://goreportcard.com/report/github.com/fractalqb/goxic)
[![GoDoc](https://godoc.org/github.com/fractalqb/goxic?status.svg)](https://godoc.org/github.com/fractalqb/goxic)

`import "git.fractalqb.de/fractalqb/goxic"`

---

Template engine with temlates that only have named placeholders
– nothing more! 

Package goxic implements the fractal[qb] toxic template engine
concept for the GO programming language.  For details on the idea
behind the toxic template engine see

  https://fractalqb.de/toxic/index.html

Short version is: A Template is nothing but static parts (text)
interspersed with placeholders. Placeholders have a unique name and
may appear several times withn a template. One cannot generate
output from a template only.  Before output can be generated –
"emitted" in goxic – all placeholders have to be bound to some
Content. The bindings are held by a separate object of type BounT
(short for "bound template"). This way a single template can be
used many times with different bindings. A bound template itself is
Content. – Be aware of infinite recursion!

One can build a template through the Template API of the goxic
packed. Though quite easy it is not very convenient to build
templates this way. It is considered to be more common to build a
template by parsing it from a file. The package provides a quite
general Parser that also supports nesting of templates. Finally one
can create new templates from bound templates where not every
placeholder has been bound to content, i.e. to Fixate a BounT.
This way one can build coarse grained templates from smaller ones
without loosing efficiency.
