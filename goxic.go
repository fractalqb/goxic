// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017-2018 Marcus Perlick
package goxic

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

type fragment []byte

type PhIdx = int

// Template holds a sequence of fixed (dstatic) content fragment that
// have to be emitted verbatim. These fragments are intermixed with
// named placeholders. The content to fill in the placeholders
// generally is computed dynamically. A placeholder can appear several
// times in different posotions within a template. A template can
// start or end either with a placeholder or with fixed content.
//
// To bind cotent to placeholders one first has to crate a bound
// template (BounT) to hold the bindings. When all placeholders ar
// bound the resulting content can be emitted.
type Template struct {
	Name       string
	fix        []fragment
	plhAt      []string
	escAt      []CntWrapper // TODO
	plhNm2Idxs map[string][]int
}

func NewTemplate(name string) *Template {
	res := Template{
		Name:       name,
		plhNm2Idxs: make(map[string][]int)}
	return &res
}

// Add adds a new piece of static content to the end of the template.
// Note that static context is merged to preceeding static content as
// long as no placholder was added before.
func (t *Template) AddFix(fixFragment []byte) *Template {
	if phnm := t.PhAt(len(t.fix)); len(phnm) > 0 {
		t.fix = append(t.fix, fixFragment)
	} else if len(fixFragment) > 0 {
		if len(t.fix) == 0 {
			t.fix = []fragment{fixFragment}
		} else {
			lFrag := t.fix[len(t.fix)-1]
			tmp := make(fragment, len(lFrag)+len(fixFragment))
			copy(tmp, lFrag)
			copy(tmp[len(lFrag):], fixFragment)
			t.fix[len(t.fix)-1] = tmp
		}
	}
	return t
}

// AddStr adds a string as static content to the end of the temlate.
func (t *Template) AddStr(str string) *Template {
	return t.AddFix(fragment(str))
}

// Placeholder adds a new placeholder to the end of the template.
func (t *Template) Ph(name string) *Template {
	idx := t.FixCount()
	if phnm := t.PhAt(idx); len(phnm) > 0 {
		t.AddFix([]byte{})
		idx++
	}
	for len(t.plhAt) < idx {
		t.plhAt = append(t.plhAt, "")
	}
	t.plhAt = append(t.plhAt, name)
	if idxs, ok := t.plhNm2Idxs[name]; ok {
		idxs = append(idxs, idx)
		t.plhNm2Idxs[name] = idxs
	} else {
		t.plhNm2Idxs[name] = []int{idx}
	}
	return t
}

func (t *Template) PhWrap(name string, wrapper CntWrapper) *Template {
	res := t.Ph(name)
	t.Wrap(wrapper, len(t.plhAt)-1)
	return res
}

func (t *Template) Wrap(wrapper CntWrapper, idxs ...int) {
	for _, idx := range idxs {
		if cap(t.escAt) <= idx {
			nesc := make([]CntWrapper, idx+1)
			copy(nesc, t.escAt)
			nesc[idx] = wrapper
			t.escAt = nesc
		} else {
			t.escAt[idx] = wrapper
		}
	}
}

// FixCount returns the number of pieces of static content in the
// template.
func (t *Template) FixCount() int {
	return len(t.fix)
}

// FixAt returns the piece of static content with the index idx
// (indices are zero-based).
func (t *Template) FixAt(idx int) []byte {
	if idx < 0 || idx >= len(t.fix) {
		return nil
	} else {
		return t.fix[idx]
	}
}

func (t *Template) ForeachPh(r func(name string, idxs []int)) {
	for nm, idxs := range t.plhNm2Idxs {
		r(nm, idxs)
	}
}

// PlaceholderNum returns the number of placeholders defined in the
// template.
func (t *Template) PhNum() int {
	return len(t.plhNm2Idxs)
}

// Placeholders returns all placeholders – more precisely placeholder
// names – defined in the template.
func (t *Template) Phs() []string {
	res := make([]string, 0, len(t.plhNm2Idxs))
	for nm := range t.plhNm2Idxs {
		res = append(res, nm)
	}
	return res
}

// PlaceholderAt returns the placeholder that will be emitted between
// static content idx-1 and static content idx. Note that
// PlaceholderAt(0) will be emitted before the first piece of static
// content. This placeholder is optional.
func (t *Template) PhAt(idx int) string {
	if t.plhAt == nil || idx >= len(t.plhAt) {
		return ""
	}
	return t.plhAt[idx]
}

func (t *Template) WrapAt(idx int) CntWrapper {
	if t.escAt == nil || idx >= len(t.escAt) {
		return nil
	}
	return t.escAt[idx]
}

// PlaceholderIdxs returns the positions in which one placeholder will
// be emitted. Note that placeholder index 0 is – if define – emitted
// before the first piece of fixed content.
func (t *Template) PhIdxs(name string) []int {
	if res, ok := t.plhNm2Idxs[name]; !ok {
		return nil
	} else if len(res) == 0 {
		delete(t.plhNm2Idxs, name)
		return nil
	} else {
		return res
	}
}

type renameErr []string

func (re renameErr) Error() string {
	switch len(re) {
	case 1:
		return fmt.Sprintf("template has no placeholder '%s'", re[0])
	case 2:
		return fmt.Sprintf("cannot rename '%s', new name '%s' already exists",
			re[0], re[1])
	default:
		return "general renaming error"
	}
}

func RenameUnknown(err error) bool {
	re, ok := err.(renameErr)
	if ok {
		return len(re) == 1
	} else {
		return false
	}
}

func RenameExists(err error) bool {
	re, ok := err.(renameErr)
	if ok {
		return len(re) == 2
	} else {
		return false
	}
}

// RenamePh renames the current placeholder to a new name. If merge is true
// the current placeholder will be renamed even if a placeholder with the
// newName already exists. Otherwise an error is retrned. Renaming a placeholder
// that does not exists also result in an error.
func (t *Template) RenamePh(current, newName string, merge bool) error {
	var cIdxs []int
	var ok bool
	if cIdxs, ok = t.plhNm2Idxs[current]; !ok {
		return renameErr{current}
	}
	if current == newName {
		return nil
	}
	if nIdxs, ok := t.plhNm2Idxs[newName]; ok && !merge {
		return renameErr{current, newName}
	} else if ok {
		cIdxs = append(cIdxs, nIdxs...)
		sort.Slice(cIdxs, func(i, j int) bool { return cIdxs[i] < cIdxs[j] })
	}
	delete(t.plhNm2Idxs, current)
	t.plhNm2Idxs[newName] = cIdxs
	for i := range t.plhAt {
		if t.plhAt[i] == current {
			t.plhAt[i] = newName
		}
	}
	return nil
}

func (t *Template) RenamePhs(merge bool, current, newNames []string) error {
	n2i := make(map[string][]int)
	for i, cn := range current {
		if cidxs, ok := t.plhNm2Idxs[cn]; !ok {
			return renameErr{cn}
		} else {
			nn := newNames[i]
			if nidxs, ok := n2i[nn]; ok {
				if !merge {
					return renameErr{cn, nn}
				}
				nidxs = append(nidxs, cidxs...)
				sort.Slice(nidxs, func(i, j int) bool { return nidxs[i] < nidxs[j] })
				n2i[nn] = nidxs
			} else {
				n2i[nn] = cidxs
			}
		}
		delete(t.plhNm2Idxs, cn)
	}
	for n, idxs := range t.plhNm2Idxs {
		n2i[n] = idxs
	}
	t.plhNm2Idxs = n2i
	for n, idxs := range n2i {
		for _, i := range idxs {
			t.plhAt[i] = n
		}
	}
	return nil
}

func StripPath(ph string) string {
	sep := strings.LastIndexByte(ph, byte(NameSep))
	if sep >= 0 {
		ph = ph[sep+1:]
	}
	return ph
}

func (t *Template) XformPhs(merge bool, x func(string) string) error {
	cur := t.Phs()
	nnm := make([]string, len(cur))
	for i := range cur {
		nnm[i] = x(cur[i])
	}
	return t.RenamePhs(merge, cur, nnm)
}

func (t *Template) Static() ([]byte, bool) {
	if t.PhNum() == 0 {
		switch t.FixCount() {
		case 0:
			return []byte{}, true
		case 1:
			return t.FixAt(0), true
		default:
			panic("template " + t.Name + " without placeholder has many fix fragments")
		}
	} else {
		return nil, false
	}
}

func (t *Template) StaticWith(fill Content) ([]byte, bool) {
	bt := t.NewInitBounT(fill, nil)
	t = bt.Fixate()
	return t.Static()
}

// Content provides the interface Emit that will write the content to
// an io.Writer.
//
// Different than the standard write methods, Emit only returns the
// number of bytes written. If an error occurs Emit should painc with
// that error. Applications are advised to use CatchEmit to switch
// back to standard (n int, err error) I/O results. This convention
// leads to less tedious content implementations in application code,
// esp. when usig nested template/bount/content.
type Content interface {
	Emit(wr io.Writer) (wrbyte int)
}

type empty int

func (e empty) Emit(wr io.Writer) int {
	return 0
}

// Constant Empty can be use as empty Content, i.e. nothing will be emitted as
// output.
const Empty empty = 0

type CntWrapper func(cnt Content) (wrapped Content)

// BounT keeps the placeholder bindings for one specific Template. Use
// NewBounT or NewInitBounT to create a binding object from a
// Template.
type BounT struct {
	tmpl *Template
	fill []Content
}

func (t *Template) NewBounT(reuse *BounT) *BounT {
	if reuse == nil {
		reuse = new(BounT)
	}
	reuse.tmpl = t
	reuse.fill = make([]Content, t.FixCount()+1)
	return reuse
}

func (t *Template) NewInitBounT(cnt Content, reuse *BounT) *BounT {
	reuse = t.NewBounT(reuse)
	for i := 0; i < len(reuse.fill); i++ {
		if len(t.PhAt(i)) > 0 {
			if esc := t.WrapAt(i); esc != nil {
				cnt = esc(cnt)
			}
			reuse.fill[i] = cnt
		}
	}
	return reuse
}

func (bt *BounT) Template() *Template {
	return bt.tmpl
}

// Method Bind returns the number of "anonymous binds", i.e. placeholders with
// empty names that got a binding.
func (bt *BounT) Bind(phIdxs []int, cnt Content) (anonymous int) {
	t := bt.Template()
	for _, i := range phIdxs {
		if len(t.PhAt(i)) == 0 {
			anonymous++
		}
		if esc := t.WrapAt(i); esc != nil {
			cnt = esc(cnt)
		}
		bt.fill[i] = cnt
	}
	return anonymous
}

func (bt *BounT) BindName(name string, cnt Content) error {
	idxs := bt.Template().PhIdxs(name)
	if idxs == nil {
		return fmt.Errorf("no placeholder: '%s'", name)
	} else {
		bt.Bind(idxs, cnt)
		return nil
	}
}

func (bt *BounT) BindIfName(name string, cnt Content) {
	idxs := bt.Template().PhIdxs(name)
	if idxs != nil {
		bt.Bind(idxs, cnt)
	}
}

func (bt *BounT) BindMatch(pattern *regexp.Regexp, cnt Content) {
	for _, ph := range bt.Template().Phs() {
		if pattern.MatchString(ph) {
			bt.BindName(ph, cnt)
		}
	}
}

type EmitError struct {
	Count int
	Err   error
}

func CatchEmit(bt *BounT, wr io.Writer) (n int, err error) {
	defer func() {
		if rek := recover(); rek != nil {
			if ee, ok := rek.(EmitError); ok {
				n = ee.Count
				err = ee.Err
			} else {
				panic(err)
			}
		}
	}()
	n = bt.Emit(wr)
	return n, nil
}

func (ee EmitError) Error() string {
	return ee.Err.Error()
}

// Method Emit panics with an EmitError when an error occures during
// emitting. Use CatchEmit() to easily get back to an io.Writer like
// error return.
func (bt *BounT) Emit(out io.Writer) (n int) {
	n = 0
	fixs := bt.tmpl.fix
	fCount := len(fixs)
	for i := 0; i < fCount; i++ {
		if f := bt.fill[i]; f != nil {
			n += f.Emit(out)
		} else if ph := bt.tmpl.PhAt(i); len(ph) > 0 {
			panic(EmitError{n,
				fmt.Errorf("unbound placeholder '%s' in template '%s'",
					ph,
					bt.tmpl.Name)})
		}
		if c, err := out.Write(fixs[i]); err != nil {
			panic(EmitError{n + c, err})
		} else {
			n += c
		}
	}
	if f := bt.fill[fCount]; f != nil {
		n += f.Emit(out)
	} else if ph := bt.tmpl.PhAt(fCount); len(ph) > 0 {
		panic(EmitError{n,
			fmt.Errorf("unbound placeholder '%s' in template '%s'",
				ph,
				bt.tmpl.Name)})
	}
	return n
}

const NameSep = ':'

func (bt *BounT) Fixate() *Template {
	it := bt.Template()
	if it.PhNum() == 0 {
		return nil
	}
	res := NewTemplate(it.Name)
	bt.fix(res, "")
	return res
}

func (bt *BounT) fix(to *Template, phPrefix string) {
	it := bt.Template()
	for idx, frag := range it.fix {
		pre := bt.fill[idx]
		if pre == nil {
			if phnm := it.PhAt(idx); len(phnm) > 0 {
				to.Ph(phPrefix + phnm)
			}
		} else if sbt, ok := pre.(*BounT); ok {
			subPrefix := phPrefix + sbt.Template().Name + string(NameSep)
			sbt.fix(to, subPrefix)
		} else {
			buf := bytes.NewBuffer(nil)
			pre.Emit(buf)
			to.AddStr(buf.String())
		}
		to.AddFix(frag)
	}
	idx := len(it.fix)
	pre := bt.fill[idx]
	if pre == nil {
		if phnm := it.PhAt(idx); len(phnm) > 0 {
			to.Ph(phPrefix + phnm)
		}
	} else if sbt, ok := pre.(*BounT); ok {
		subPrefix := phPrefix + sbt.Template().Name + string(NameSep)
		sbt.fix(to, subPrefix)
	} else {
		buf := bytes.NewBuffer(nil)
		pre.Emit(buf)
		to.AddStr(buf.String())
	}
}

func (bt *BounT) Wrap(wrapper func(Content) Content) {
	for i, f := range bt.fill {
		if f != nil {
			w := wrapper(f)
			bt.fill[i] = w
		}
	}
}

type tagMode int

//go:generate -t tagMode
const (
	tagNone tagMode = iota
	tagMand
	tagOpt
	tagIgnore
)

func parseTag(tag string) (mode tagMode, placeholder string, err error) {
	mode = tagMand
	if len(tag) > 0 {
		if sep := strings.IndexRune(tag, ' '); sep >= 0 {
			if sep == 0 {
				return mode, "", fmt.Errorf("goxic imap: tag format '%s'", tag)
			}
			placeholder = tag[:sep]
			if placeholder == "-" {
				return tagIgnore, placeholder, nil
			}
			switch tag[sep+1:] {
			case "opt":
				mode = tagOpt
			default:
				return mode, "", fmt.Errorf("goxic imap: illgeal tag option '%s'", tag[sep+1:])
			}
			return mode, placeholder, nil
		} else {
			return mode, tag, nil
		}
	} else {
		return tagNone, "", nil
	}
}
