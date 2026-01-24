package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/daved/clic"
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

type ListWeight struct {
	out   io.Writer
	db    *db.Queries
	limit int64
}

func NewListWeight(out io.Writer, db *db.Queries) *ListWeight {
	return &ListWeight{
		out:   out,
		db:    db,
		limit: 7,
	}
}

func (lw *ListWeight) ListWeights() clic.HandlerFunc {
	return func(ctx context.Context) error {
		weights, err := lw.db.ListWeights(ctx, lw.limit)
		if err != nil {
			return err
		}

		if len(weights) == 0 {
			fmt.Fprintf(lw.out, "no weight entries found\n")
			return nil
		}

		for _, w := range weights {
			fmt.Fprintf(lw.out, "%s: %d lbs\n", w.Date, w.Pounds)
		}
		return nil
	}
}

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
