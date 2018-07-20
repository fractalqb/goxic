package goxic

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	BftMarker  = "$"
	BftPathSep = "."
)

var zero = reflect.Value{}

func bftResolve(path string, data interface{}) (bindThis interface{}, err error) {
	psegs := strings.Split(path, BftPathSep)
	for si, seg := range psegs {
		rval := reflect.ValueOf(data)
		if idx, err := strconv.Atoi(seg); err == nil {
			switch rval.Type().Kind() {
			case reflect.Array, reflect.Slice:
				if idx < 0 {
					idx = rval.Len() + idx
				}
				if idx < 0 || idx >= rval.Len() {
					return nil, nil
				}
				data = rval.Index(idx).Interface()
			default:
				return nil, fmt.Errorf("segemnt %d in path '%s' requires slice or array, got %s",
					si,
					path,
					rval.Type().Kind())
			}
		} else {
			switch rval.Type().Kind() {
			case reflect.Map:
				tmp := rval.MapIndex(reflect.ValueOf(seg))
				if tmp == zero {
					return nil, nil
				} else {
					data = tmp.Interface()
				}
			case reflect.Struct:
				tmp := rval.FieldByName(seg)
				if tmp == zero {
					return nil, nil
				} else {
					data = tmp.Interface()
				}
			default:
				return nil, fmt.Errorf("segemnt %d in path '%s' requires map or struct, got %s",
					si,
					path,
					rval.Type().Kind())
			}
		}
	}
	return data, nil
}

func bftSplitSpec(specPh string) (fmt string, path string) {
	sep := strings.Index(specPh, " ")
	if sep > 0 {
		return specPh[:sep], specPh[sep+1:]
	}
	return "", specPh
}

func (bt *BounT) Fill(data interface{}, overwrite bool) (missed int, err error) {
	tpl := bt.Template()
	for ph, idxs := range tpl.plhNm2Idxs {
		if !strings.HasPrefix(ph, BftMarker) {
			continue
		}
		ph := ph[1:]
		// TODO maybe its efficient to 1st check if there is something to bind
		//      consider overwrite
		fmt, path := bftSplitSpec(ph)
		bv, err := bftResolve(path, data) // TODO slow?
		if err != nil {
			return -1, err
		}
		if bv == nil {
			missed++
		} else if len(fmt) == 0 {
			bt.BindP(idxs, bv)
		} else {
			bt.BindFmt(idxs, fmt, bv)
		}
	}
	return missed, nil
}
