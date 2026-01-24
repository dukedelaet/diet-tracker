package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/guitarkeegan/diet-tracker/internal/db"
)

type weight struct {
	out    io.Writer
	db     *db.Queries
	pounds int64
}

var (
	alreadyLoggedWeightErr = errors.New("weight has already been logged today")
	notFoundErr            = errors.New("not found")
)

func NewWeight(out io.Writer, db *db.Queries) *weight {
	return &weight{
		out: out,
		db:  db,
	}
}

func (w *weight) HandleCommand(ctx context.Context) error {

	curDate := time.Now().In(time.Local).Format("2006-01-02")
	if w.pounds <= 0 {
		weight, err := w.db.GetWeightByDate(ctx, curDate)
		if err != nil {
			return err
		}
		fmt.Fprintf(w.out, "%d\n", weight.Pounds)
		return nil
	}
	weightHist, err := w.db.GetWeightByDate(ctx, curDate)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return notFoundErr
		}
	}
	if weightHist.Pounds > 0 {
		return alreadyLoggedWeightErr
	}
	curWeight := db.CreateWeightParams{
		Date:   curDate,
		Pounds: w.pounds,
	}
	wei, err := w.db.CreateWeight(ctx, curWeight)
	if err != nil {
		return err
	}

	fmt.Fprintf(w.out, "successfully logged weight: %v", wei)

	return nil
}
