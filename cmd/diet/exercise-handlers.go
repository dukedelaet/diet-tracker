package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/daved/clic"
	"github.com/guitarkeegan/diet-tracker/internal/db"
)

type Exercise struct {
	out io.Writer
}

func NewExercise(out io.Writer) *Exercise {
	return &Exercise{out: out}
}

func (e *Exercise) HandleCommand(ctx context.Context) error {
	return nil
}

type CreateExercise struct {
	out          io.Writer
	db           *db.Queries
	exerciseType string
	duration     int64
}

func NewCreateExercise(out io.Writer, db *db.Queries) *CreateExercise {
	return &CreateExercise{
		out: out,
		db:  db,
	}
}

func (ce *CreateExercise) HandleCommand(ctx context.Context) error {
	if ce.exerciseType == "" {
		return fmt.Errorf("exercise type is required (cardio or strength)")
	}
	if ce.exerciseType != "cardio" && ce.exerciseType != "strength" {
		return fmt.Errorf("exercise type must be 'cardio' or 'strength'")
	}
	if ce.duration <= 0 {
		return fmt.Errorf("duration must be greater than 0")
	}

	curDate := time.Now().In(time.Local).Format(time.DateOnly)
	exercise, err := ce.db.CreateExercise(ctx, db.CreateExerciseParams{
		Date:         curDate,
		ExerciseType: ce.exerciseType,
		Duration:     ce.duration,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ce.out, "logged %s for %d min on %s\n", exercise.ExerciseType, exercise.Duration, exercise.Date)
	return nil
}

type ListExercise struct {
	out   io.Writer
	db    *db.Queries
	limit int64
}

func NewListExercise(out io.Writer, db *db.Queries) *ListExercise {
	return &ListExercise{
		out:   out,
		db:    db,
		limit: 10,
	}
}

func (le *ListExercise) ListExercises() clic.HandlerFunc {
	return func(ctx context.Context) error {
		exercises, err := le.db.ListExercises(ctx, le.limit)
		if err != nil {
			return err
		}

		if len(exercises) == 0 {
			fmt.Fprintf(le.out, "no exercises found\n")
			return nil
		}

		for _, e := range exercises {
			fmt.Fprintf(le.out, "%s: %s %d min\n", e.Date, e.ExerciseType, e.Duration)
		}
		return nil
	}
}
