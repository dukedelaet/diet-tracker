package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/daved/clic"
	"github.com/guitarkeegan/diet-tracker/internal/db"
)

var (
	noMacrosErr              = errors.New("protein, carbs, and fat cannot all be 0")
	missingNameErr           = errors.New("meal must have a name")
	mealNameAlreadyExistsErr = errors.New("meal name already exists")
)

type Macros struct {
	Protein int64
	Carbs   int64
	Fat     int64
	Cals    int64
}

type Meal struct {
	out io.Writer
}

type CreateMeal struct {
	out    io.Writer
	db     *db.Queries
	name   string
	macros Macros
}

type ListMeal struct {
	out   io.Writer
	db    *db.Queries
	limit int64
}

type LogMeal struct {
	out  io.Writer
	db   *db.Queries
	name string
}

func (lm *LogMeal) LogMeal() clic.HandlerFunc {
	return func(ctx context.Context) error {
		if lm.name == "" {
			return fmt.Errorf("must provide a meal name\n")
		}

		curDate := time.Now().In(time.Local).Format(time.DateOnly)
		lm.db.LogMealByName(ctx, db.LogMealByNameParams{
			Date: curDate,
			Name: lm.name,
		})

		return nil
	}
}

func NewLogMeal(out io.Writer, db *db.Queries) *LogMeal {
	return &LogMeal{
		out: out,
		db:  db,
	}
}

func NewListMeal(out io.Writer, db *db.Queries) *ListMeal {
	return &ListMeal{
		out:   out,
		db:    db,
		limit: 10,
	}
}

func (lm *ListMeal) ListMeals() clic.HandlerFunc {
	return func(ctx context.Context) error {
		meals, err := lm.db.ListMeals(ctx, lm.limit)
		if err != nil {
			return err
		}

		for _, meal := range meals {
			fmt.Fprintf(lm.out, "%s\nProtein: %d, Carbs: %d, Fat: %d, Cals: %d\n--\n", meal.Name, meal.Protein, meal.Carbs, meal.Fat, meal.Calories)
		}

		return nil
	}
}

func NewCreateMeal(out io.Writer, db *db.Queries) *CreateMeal {
	return &CreateMeal{
		out:    out,
		db:     db,
		name:   "",
		macros: Macros{},
	}
}

func (cm *CreateMeal) CalculateCals() int64 {
	return cm.macros.Carbs*4 + cm.macros.Protein*4 + cm.macros.Fat*9
}

func (cm *CreateMeal) hasMacros(p, c, f int64) bool {
	if p > 0 || c > 0 || f > 0 {
		return true
	}
	return false
}

func (cm *CreateMeal) validateCreateMeal() error {

	if !cm.hasMacros(cm.macros.Protein, cm.macros.Carbs, cm.macros.Fat) {
		return noMacrosErr
	}

	if cm.name == "" {
		return missingNameErr
	}

	return nil
}

func (cm *CreateMeal) HandleCommand(ctx context.Context) error {

	err := cm.validateCreateMeal()
	if err != nil {
		return err
	}

	meal, err := cm.db.GetMealByName(ctx, cm.name)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if meal.Name != "" {
		return mealNameAlreadyExistsErr
	}

	newMeal, err := cm.db.CreateMeal(ctx, db.CreateMealParams{
		Name:     cm.name,
		Protein:  cm.macros.Protein,
		Carbs:    cm.macros.Carbs,
		Fat:      cm.macros.Fat,
		Calories: cm.CalculateCals(),
	})
	if err != nil {
		return err
	}

	jsonFmt, err := json.MarshalIndent(newMeal, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintf(cm.out, "%s\n", jsonFmt)

	return nil
}

func NewMeal(out io.Writer, db *db.Queries) *Meal {
	return &Meal{
		out: out,
	}
}
func (m *Meal) HandleCommand(ctx context.Context) error {
	return nil
}
