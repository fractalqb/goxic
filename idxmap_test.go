package goxic

import (
	"testing"

	"github.com/stvp/assert"
)

type IMap struct {
	*Template
	Foo []int // use naming convention to map, same as `goxic:"Foo"`
	Bar []int `goxic:"bar"`
	Baz []int `goxic:"baz opt"`
}

func TestIndexMap(t *testing.T) {
	tmpl := NewTemplate(t.Name())
	tmpl.Ph("Foo")
	tmpl.Ph("bar")
	tmpl.Ph("quux")
	var imap IMap
	unmappend := InitIndexMap(&imap, tmpl, IdName)
	assert.Equal(t, tmpl, imap.Template)
	assertIndices(t, imap.Foo, 0)
	assertIndices(t, imap.Bar, 1)
	assertIndices(t, imap.Baz)
	assert.Equal(t, 1, len(unmappend.Placeholders))
	assert.Equal(t, "quux", unmappend.Placeholders[0])
}
