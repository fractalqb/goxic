// Template engine that only has named placeholders – nothing more!
// Copyright (C) 2017 Marcus Perlick
package goxic

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Parser struct {
	StartInlinePh    string
	EndInlinePh      string
	BlockPh          *regexp.Regexp
	PhNameRgxGrp     int
	PhLBrkRgxGrp     int
	PhTBrkRgxGrp     int
	StartSubTemplate *regexp.Regexp
	StartNameRgxGrp  int
	StartLBrkRgxGrp  int
	EndSubTemplate   *regexp.Regexp
	EndNameRgxGrp    int
	EndTBrkRgxGrp    int
	Endl             string
	PrepLine         func(string) string
}

func PrepTrimWS(line string) (trimmed string) {
	return strings.Trim(line, " \t")
}

func pPush(stack []string, s string) (updStack []string, path string) {
	updStack = append(stack, s)
	path = pathStr(updStack)
	return updStack, path
}

func top(stack []string) string {
	if len(stack) == 0 {
		panic("empty stack")
	} else {
		return stack[len(stack)-1]
	}
}

func pPop(stack []string) (updStack []string, path string) {
	updStack = stack[:len(stack)-1]
	path = pathStr(updStack)
	return updStack, path
}

const pathSep = '/'

func pathStr(path []string) string {
	buf := bytes.NewBufferString("")
	for i, s := range path {
		if i > 0 {
			buf.WriteRune(pathSep)
		}
		buf.WriteString(s)
	}
	return buf.String()
}

func tmplName(rootName string, path string) string {
	if len(path) > 0 {
		return rootName + string(pathSep) + path
	} else {
		return rootName
	}
}

func needTemplate(t *Template, rootPath string, name string) (*Template, error) {
	name = tmplName(rootPath, name)
	if t == nil {
		t = NewTemplate(name)
	} else if t.Name != name {
		return t, fmt.Errorf("template name mismatch '%s' ≠ '$s'",
			t.Name,
			name)
	}
	return t, nil
}

func storeTemplate(res map[string]*Template, t *Template, key string, dup DuplicateTemplates) bool {
	if t == nil {
		return true
	}
	if old, ok := res[key]; ok && old != t {
		dup[key] = t
		return false
	} else {
		res[key] = t
		return true
	}
}

func (p *Parser) phLBrk(match []string) bool {
	return len(match[p.PhLBrkRgxGrp]) == 0
}

func (p *Parser) phTBrk(match []string) bool {
	return len(match[p.PhTBrkRgxGrp]) == 0
}

func (p *Parser) startLBrk(match []string) bool {
	return len(match[p.StartLBrkRgxGrp]) == 0
}

func (p *Parser) endTBrk(match []string) bool {
	return len(match[p.EndTBrkRgxGrp]) == 0
}

type DuplicateTemplates map[string]*Template

func (err DuplicateTemplates) Error() string {
	buf := bytes.NewBufferString("duplicate templates: ")
	sep := ""
	for nm, _ := range err {
		buf.WriteString(sep)
		buf.WriteString(nm)
		sep = ", "
	}
	return buf.String()
}

func (p *Parser) Parse(rd io.Reader, rootName string, into map[string]*Template) error {
	var dup DuplicateTemplates = make(map[string]*Template)
	scn := bufio.NewScanner(rd)
	path := []string{}
	pStr := ""
	endl := ""
	var curTmpl *Template = nil
	for scn.Scan() {
		line := scn.Text()
		if match := p.StartSubTemplate.FindStringSubmatch(line); len(match) > 0 {
			if p.startLBrk(match) {
				var err error
				curTmpl, err = needTemplate(curTmpl, rootName, pStr)
				if err != nil {
					return err
				}
				curTmpl.AddStr(endl)
			}
			storeTemplate(into, curTmpl, pStr, dup)
			subtName := match[p.StartNameRgxGrp]
			if strings.IndexRune(subtName, pathSep) >= 0 {
				return fmt.Errorf(
					"sub-temlpate name '%s' contains path separator %c",
					subtName,
					pathSep)
			}
			path, pStr = pPush(path, subtName)
			curTmpl = into[pStr]
			endl = ""
		} else if match := p.EndSubTemplate.FindStringSubmatch(line); len(match) > 0 {
			subtName := match[p.EndNameRgxGrp]
			if len(path) == 0 || top(path) != subtName {
				return fmt.Errorf(
					"unexpected sub-template end '%s'",
					subtName)
			}
			storeTemplate(into, curTmpl, pStr, dup)
			path, pStr = pPop(path)
			curTmpl = into[pStr]
			if p.endTBrk(match) {
				endl = p.Endl
			} else {
				endl = ""
			}
		} else if match := p.BlockPh.FindStringSubmatch(line); len(match) > 0 {
			var err error
			curTmpl, err = needTemplate(curTmpl, rootName, pStr)
			if err != nil {
				return err
			}
			if p.phLBrk(match) {
				curTmpl.AddStr(endl)
			}
			phName := match[p.PhNameRgxGrp]
			curTmpl.Placeholder(phName)
			if p.phTBrk(match) {
				endl = p.Endl
			} else {
				endl = ""
			}
		} else {
			var err error
			curTmpl, err = needTemplate(curTmpl, rootName, pStr)
			if err != nil {
				return err
			}
			curTmpl.AddStr(endl)
			if p.PrepLine != nil {
				line = p.PrepLine(line)
			}
			p.addLine(curTmpl, line)
			endl = p.Endl
		}
	}
	if len(path) > 0 {
		return fmt.Errorf("end of input in nested template")
	}
	storeTemplate(into, curTmpl, pStr, dup)
	if len(dup) > 0 {
		return dup
	} else {
		return nil
	}
}

func (p *Parser) addLine(t *Template, line string) error {
	for tok := strings.Index(line, p.StartInlinePh); tok >= 0; tok = strings.Index(line, p.StartInlinePh) {
		if tok > 0 {
			t.AddStr(line[:tok])
			line = line[tok+len(p.StartInlinePh):]
		} else if tok == 0 {
			line = line[len(p.StartInlinePh):]
		}
		tok = strings.Index(line, p.EndInlinePh)
		if tok < 0 {
			return fmt.Errorf(
				"unexpected end of line in placeholder '%s'",
				line)
		}
		t.Placeholder(line[:tok])
		line = line[tok+len(p.EndInlinePh):]
	}
	if len(line) > 0 {
		t.AddStr(line)
	}
	return nil
}
