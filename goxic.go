package goxic

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

type fragment []byte

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

// FixAt returns the peice of static content with the index idx
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

// Retuns all placeholders – more precisely placeholder names –
// defined in the template.
func (t *Template) Placeholders() []string {
	res := make([]string, 0, len(t.plhNm2Idxs))
	for nm := range t.plhNm2Idxs {
		res = append(res, nm)
	}
	return res
}

// PlaceholderAt returns the placeholder that will be emitted between
// static content idx-1 and static content idx. Note that
// PlaceholderAt(0) will be emitted before the first peice of static
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

func (t *Template) NewBounT() *BounT {
	res := BounT{
		tmpl: t,
		fill: make([]Content, t.FixCount()+1)}
	return &res
}

func (t *Template) NewInitBounT(cnt Content) *BounT {
	res := t.NewBounT()
	for i := 0; i < len(res.fill); i++ {
		if len(t.PlaceholderAt(i)) > 0 {
			res.fill[i] = cnt
		}
	}
	return res
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

func (bt *BounT) Fixate() (*Template, error) {
	it := bt.Template()
	if it.PlaceholderNum() == 0 {
		return nil, nil
	}
	res := NewTemplate(it.Name)
	if err := bt.fix(res, ""); err != nil {
		return nil, err
	}
	return res, nil
}

func (bt *BounT) fix(to *Template, phPrefix string) error {
	it := bt.Template()
	for idx, frag := range it.fix {
		pre := bt.fill[idx]
		if pre == nil {
			if phnm := it.PlaceholderAt(idx); len(phnm) > 0 {
				to.Placeholder(phPrefix + phnm)
			}
		} else if sbt, ok := pre.(*BounT); ok {
			subPrefix := phPrefix + it.Name + string(pathSep)
			if err := sbt.fix(to, subPrefix); err != nil {
				return err
			}
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
		subPrefix := phPrefix + it.Name + string(pathSep)
		if err := sbt.fix(to, subPrefix); err != nil {
			return err
		}
	} else {
		buf := bytes.NewBuffer(nil)
		pre.Emit(buf)
		to.AddStr(buf.String())
	}
	return nil
}

func (bt *BounT) Wrap(wrapper func(Content) Content) {
	for i, f := range bt.fill {
		if f != nil {
			w := wrapper(f)
			bt.fill[i] = w
		}
	}
}

func parseTag(tag string) (placeholder string, optional bool, err error) {
	optional = false
	if len(tag) > 0 {
		if sep := strings.IndexRune(tag, ' '); sep >= 0 {
			if sep == 0 {
				return "", false, fmt.Errorf("goxic imap: tag format '%s'", tag)
			}
			placeholder = tag[:sep]
			switch tag[sep+1:] {
			case "opt":
				optional = true
			default:
				return "", false, fmt.Errorf("goxic imap: illgeal tag option '%s'", tag[sep+1:])
			}
			return placeholder, optional, nil
		} else {
			return tag, optional, nil
		}
	} else {
		return "", false, nil
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

func InitIndexMap(imap interface{}, tmpl *Template) *Unmapped {
	imTy := reflect.TypeOf(imap).Elem()
	im := reflect.ValueOf(imap).Elem()
	mappedPhs := make(map[string]bool)
	switch imTy.Kind() {
	case reflect.Struct:
		for fidx := 0; fidx < imTy.NumField(); fidx++ {
			sfTy := imTy.Field(fidx)
			if sfTy.Anonymous && sfTy.Type == reflect.TypeOf(tmpl) {
				imapVal := reflect.ValueOf(imap).Elem()
				imapVal.Field(fidx).Set(reflect.ValueOf(tmpl))
			} else {
				// TODO skip unsettable fields
				ph, opt, err := parseTag(sfTy.Tag.Get("goxic"))
				if err != nil {
					panic("cannot make index map: " + err.Error())
				}
				if len(ph) == 0 {
					continue
				}
				var idxs []int
				idxs = tmpl.PlaceholderIdxs(string(ph))
				if idxs != nil {
					mappedPhs[ph] = true
					sf := im.Field(fidx)
					sf.Set(reflect.ValueOf(idxs))
				} else if opt {
					sf := im.Field(fidx)
					sf.Set(reflect.ValueOf(emptyIndices))
				}
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

func MustIndexMap(imap interface{}, t *Template) {
	missing := InitIndexMap(imap, t)
	if missing != nil {
		panic(missing)
	}
}

func MapAll(unmapped *Unmapped) {
	if unmapped != nil {
		panic(unmapped)
	}
}
