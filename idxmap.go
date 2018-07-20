package goxic

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

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
				var idxs []int = tmpl.PhIdxs(string(ph))
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
	if len(mappedPhs) != tmpl.PhNum() {
		um := &Unmapped{T: tmpl}
		for _, p := range tmpl.Phs() {
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

var emptyIndices = []int{}

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
