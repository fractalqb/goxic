// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package goxic

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
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
func (t *Template) AddFix(fixFragment []byte) {
	if phnm := t.PlaceholderAt(len(t.fix)); len(phnm) > 0 {
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
}

// AddStr adds a string as static content to the end of the temlate.
func (t *Template) AddStr(str string) {
	t.AddFix(fragment(str))
}

// Placeholder adds a new placeholder to the end of the template.
func (t *Template) Placeholder(name string) {
	idx := t.FixCount()
	if phnm := t.PlaceholderAt(idx); len(phnm) > 0 {
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

func (t *Template) ForeachPlaceholder(r func(name string, idxs []int)) {
	for nm, idxs := range t.plhNm2Idxs {
		r(nm, idxs)
	}
}

// PlaceholderNum returns the number of placeholders defined in the
// template.
func (t *Template) PlaceholderNum() int {
	return len(t.plhNm2Idxs)
}

// Placeholders returns all placeholders – more precisely placeholder
// names – defined in the template.
func (t *Template) Placeholders() []string {
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
func (t *Template) PlaceholderAt(idx int) string {
	if t.plhAt == nil || idx >= len(t.plhAt) {
		return ""
	} else if name := t.plhAt[idx]; name == "" {
		return ""
	} else {
		return name
	}
}

// PlaceholderIdxs returns the positions in which one placeholder will
// be emitted. Note that placeholder index 0 is – if define – emitted
// before the first piece of fixed content.
func (t *Template) PlaceholderIdxs(name string) []int {
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
		sort.Slice(cIdxs, func(i, j int) bool {
			return cIdxs[i] < cIdxs[j]
		})
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

func (t *Template) XformPhs(merge bool, x func(string) string) error {
	phs := t.Placeholders()
	for _, ph := range phs {
		if err := t.RenamePh(ph, x(ph), merge); err != nil {
			return err
		}
	}
	return nil
}

func (t *Template) Static() ([]byte, bool) {
	if t.PlaceholderNum() == 0 {
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
		if len(t.PlaceholderAt(i)) > 0 {
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
	for _, i := range phIdxs {
		if len(bt.Template().PlaceholderAt(i)) == 0 {
			anonymous++
		}
		bt.fill[i] = cnt
	}
	return anonymous
}

func (bt *BounT) BindName(name string, cnt Content) error {
	idxs := bt.Template().PlaceholderIdxs(name)
	if idxs == nil {
		return fmt.Errorf("no placeholder: '%s'", name)
	} else {
		bt.Bind(idxs, cnt)
		return nil
	}
}

func (bt *BounT) BindIfName(name string, cnt Content) {
	idxs := bt.Template().PlaceholderIdxs(name)
	if idxs != nil {
		bt.Bind(idxs, cnt)
	}
}

func (bt *BounT) BindMatch(pattern *regexp.Regexp, cnt Content) {
	for _, ph := range bt.Template().Placeholders() {
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
		} else if ph := bt.tmpl.PlaceholderAt(i); len(ph) > 0 {
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
	} else if ph := bt.tmpl.PlaceholderAt(fCount); len(ph) > 0 {
		panic(EmitError{n,
			fmt.Errorf("unbound placeholder '%s' in template '%s'",
				ph,
				bt.tmpl.Name)})
	}
	return n
}

func (bt *BounT) Fixate() *Template {
	it := bt.Template()
	if it.PlaceholderNum() == 0 {
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
			if phnm := it.PlaceholderAt(idx); len(phnm) > 0 {
				to.Placeholder(phPrefix + phnm)
			}
		} else if sbt, ok := pre.(*BounT); ok {
			subPrefix := phPrefix + sbt.Template().Name + ":" //string(pathSep)
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
		if phnm := it.PlaceholderAt(idx); len(phnm) > 0 {
			to.Placeholder(phPrefix + phnm)
		}
	} else if sbt, ok := pre.(*BounT); ok {
		subPrefix := phPrefix + sbt.Template().Name + ":" //string(pathSep)
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

var emptyIndices = []int{}

type Unmapped struct {
	T            *Template
	Placeholders []string
}

func (u *Unmapped) Error() string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf,
		"unmapped placeholders in template '%s': %s",
		u.T.Name,
		strings.Join(u.Placeholders, ", "))
	return buf.String()
}

func IdName(nm string) string { return nm }

func isPhIdxs(f *reflect.StructField,
	mapNames func(string) string) (ph string, opt bool, err error) {
	// TODO skip unsettable fields
	mode, ph, err := parseTag(f.Tag.Get("goxic"))
	switch mode {
	case tagMand:
		opt = false
	case tagOpt:
		opt = true
	}
	if err != nil {
		return "", opt, err
	}
	if len(ph) == 0 && mapNames != nil {
		ph = mapNames(f.Name)
	}
	return ph, opt, nil
}

func InitIndexMap(imap interface{}, tmpl *Template, mapNames func(string) string) *Unmapped {
	imTy := reflect.TypeOf(imap).Elem()
	im := reflect.ValueOf(imap).Elem()
	mappedPhs := make(map[string]bool)
	switch imTy.Kind() {
	case reflect.Struct:
		for fidx := 0; fidx < imTy.NumField(); fidx++ {
			sfTy := imTy.Field(fidx)
			if sfTy.Anonymous && sfTy.Type == reflect.TypeOf(tmpl) {
				// TODO at most once!
				imapVal := reflect.ValueOf(imap).Elem()
				imapVal.Field(fidx).Set(reflect.ValueOf(tmpl))
			} else if ph, opt, err := isPhIdxs(&sfTy, mapNames); len(ph) > 0 {
				var idxs []int = tmpl.PlaceholderIdxs(string(ph))
				if idxs != nil {
					mappedPhs[ph] = true
					sf := im.Field(fidx)
					sf.Set(reflect.ValueOf(idxs))
				} else if opt {
					sf := im.Field(fidx)
					sf.Set(reflect.ValueOf(emptyIndices))
				} // TODO error on missing mandatory indexes
			} else if err != nil {
				panic("failed to index field: " + err.Error())
			}
		}
	default:
		panic("cannto make index map in " + imTy.Kind().String())
	}
	if len(mappedPhs) != tmpl.PlaceholderNum() {
		um := &Unmapped{T: tmpl}
		for _, p := range tmpl.Placeholders() {
			if _, ok := mappedPhs[p]; !ok {
				um.Placeholders = append(um.Placeholders, p)
			}
		}
		return um
	} else {
		return nil
	}
}

func MustIndexMap(imap interface{}, t *Template, mapNames func(string) string) {
	missing := InitIndexMap(imap, t, mapNames)
	if missing != nil {
		panic(missing)
	}
}

func MapAll(unmapped *Unmapped) {
	if unmapped != nil {
		panic(unmapped)
	}
}
