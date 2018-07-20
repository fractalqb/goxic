# goxic
[![Build Status](https://travis-ci.org/fractalqb/goxic.svg)](https://travis-ci.org/fractalqb/goxic)
[![codecov](https://codecov.io/gh/fractalqb/goxic/branch/master/graph/badge.svg)](https://codecov.io/gh/fractalqb/goxic)
[![Go Report Card](https://goreportcard.com/badge/github.com/fractalqb/goxic)](https://goreportcard.com/report/github.com/fractalqb/goxic)
[![GoDoc](https://godoc.org/github.com/fractalqb/goxic?status.svg)](https://godoc.org/github.com/fractalqb/goxic)

`import "git.fractalqb.de/fractalqb/goxic"`

---
# Intro

Template engine with templates that only have named placeholders
– but no logic! 

Package goxic implements the fractal[qb] toxic template engine
concept for the [Go programming](https://golang.org) language.
For details on the idea behind the toxic template engine see

  https://fractalqb.de/toxic/index.html (TL;DR)

Short version is: A [`Template`](https://godoc.org/github.com/fractalqb/goxic#Template) is nothing but static parts (text)
interspersed with placeholders. Placeholders have a unique name and
may appear several times within a template. One cannot generate
output from a template only.  Before output can be generated –
aka "emitted" – all placeholders have to be bound to some
[_Content_](https://godoc.org/github.com/fractalqb/goxic#Content). The
bindings are held by a separate object of type
[`BounT`](https://godoc.org/github.com/fractalqb/goxic#BounT)
(short for "bound template"). This is important to be able to use a 
single template with different bindings. _Note_: A bound template 
itself [is Content](https://godoc.org/github.com/fractalqb/goxic#BounT.Emit).
 – Be aware of infinite recursion! By the way, a function that writes output
[can be content too](https://godoc.org/github.com/fractalqb/goxic#BounT.BindGen).
This has quite some implications.

One can build a template through the Template API of the goxic
packed. Though quite easy it is not very convenient to build
templates this way. It is considered to be more common to build a
template by parsing it from a file. The package provides a quite
general Parser that also supports nesting of templates. Finally one
can create new templates from bound templates where not every
placeholder has been bound to content, i.e. to _Fixate_ a `BounT`.
This way one can build coarse grained templates from smaller ones
without loosing efficiency.

The concepts described so far put a lot of control into the hands
of the programmer. But there are things that might also be helpful,
when controlled by the template writer:

1. **Escaping:** This is a common problem with e.g. HTML templates
   where injection of content can result in security breaches.

2. **Selecting and Formatting Content:** This is addressed in goxic with the
   _Bind From Template_ (BFT) feature.

Both features can be used but don't need to – goxic has a layered design. What
goxic still does ban from templates are control structures like conditionals
and loops. Such things tend to be better be placed into code of a real
programming language.

# Escaping Content

This is a common problem with e.g. HTML templates where injection of content
can result in security breaches. Here the situation is twofold: It depends on
the position of the placeholder in the template whether or how content has to
be escaped. This cannot be judged from the programmer's view. Sometimes
**ToDo…**

# Bind From Template

**ToDo…**