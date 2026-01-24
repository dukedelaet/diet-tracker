package main

import (
	"context"
	"fmt"
	"io"
)

type dietRoot struct {
	out   io.Writer
	info  string
	value string
}

func (dr *dietRoot) HandleCommand(ctx context.Context) error {
	fmt.Fprintf(dr.out, "info: %s\nvalue: %s\n", dr.info, dr.value)
	return nil
}

func NewDietRoot(out io.Writer) *dietRoot {
	return &dietRoot{
		out:  out,
		info: "root cmd",
	}
}

