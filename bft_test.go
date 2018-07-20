package goxic

import (
	"fmt"
	"os"
)

type bftAddr struct {
	Street string
	No     int
}

type bftData struct {
	Name  string
	Addrs []bftAddr
}

func ExampleBounT_Fill() {
	tmpl := NewTemplate("bft example").
		AddStr("Name: ").Ph("$Name").
		AddStr("\nLast Address: ").Ph("$Addrs.-1.Street").
		AddStr(" ").Ph("$%05d Addrs.-1.No")
	data := bftData{
		Name: "John Doe",
		Addrs: []bftAddr{
			bftAddr{Street: "Cansas Lane", No: 1},
			bftAddr{Street: "Yellow-Brick-Road", No: 33},
		},
	}
	bt := tmpl.NewBounT(nil)
	miss, err := bt.Fill(data, true)
	if err != nil {
		panic(err)
	}
	if miss != 0 {
		fmt.Printf("missed %d placeholders\n", miss)
	}
	_, err = CatchEmit(bt, os.Stdout)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// Name: John Doe
	// Last Address: Yellow-Brick-Road 00033
}
