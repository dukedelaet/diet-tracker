package main

import (
	"context"
	"io"
)

type dietRoot struct {
	out io.Writer
}

func (dr *dietRoot) HandleCommand(ctx context.Context) error {
	return nil
}

func NewDietRoot(out io.Writer) *dietRoot {
	return &dietRoot{
		out: out,
	}
}
