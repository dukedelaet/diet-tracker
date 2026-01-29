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
	out     io.Writer
	db      *db.Queries
	name    string
	portion int64 // a percentage of the meal
}

func (lm *LogMeal) LogMeal() clic.HandlerFunc {
	return func(ctx context.Context) error {
		if lm.name == "" {
			return fmt.Errorf("must provide a meal name")
		}

		curDate := time.Now().In(time.Local).Format(time.DateOnly)

		if lm.portion > 0 {
			percent := float64(lm.portion) / 100.0
			mealToLog, err := lm.db.GetMealByName(ctx, lm.name)
			if err != nil {
				return fmt.Errorf("could not find meal name: %s, error: %w", lm.name, err)
			}

			protein := int64(float64(mealToLog.Protein) * percent)
			carbs := int64(float64(mealToLog.Carbs) * percent)
			fat := int64(float64(mealToLog.Fat) * percent)

			mealLog, err := lm.db.LogMealWithPortions(ctx, db.LogMealWithPortionsParams{
				MealID:   mealToLog.ID,
				Protein:  protein,
				Carbs:    carbs,
				Fat:      fat,
				Calories: CalculateCals(protein, carbs, fat),
				Date:     curDate,
			})
			if err != nil {
				return fmt.Errorf("logMealWithPortions: Meal: %s, error: %w", lm.name, err)
			}

			// Update last_logged
			err = lm.db.UpdateMealLastLogged(ctx, db.UpdateMealLastLoggedParams{
				LastLogged: curDate,
				Name:       lm.name,
			})
			if err != nil {
				return fmt.Errorf("failed to update meal last_logged: %q, %w", lm.name, err)
			}

			fmt.Fprintf(lm.out, "logged meal %q for %s at %d%% portion (log id: %d)\n",
				lm.name, mealLog.Date, lm.portion, mealLog.ID)
			return nil
		}

		mealLog, err := lm.db.LogMeal(ctx, db.LogMealParams{
			Date: curDate,
			Name: lm.name,
		})
		if err != nil {
			return fmt.Errorf("failed to log meal %q: %w", lm.name, err)
		}

		err = lm.db.UpdateMealLastLogged(ctx, db.UpdateMealLastLoggedParams{
			LastLogged: curDate,
			Name:       lm.name,
		})
		if err != nil {
			return fmt.Errorf("failed to update meal last_logged: %q, %w", lm.name, err)
		}

		fmt.Fprintf(lm.out, "logged meal %q for %s (log id: %d)\n", lm.name, mealLog.Date, mealLog.ID)
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

// Helper to return the approximate cals based on the macros
func CalculateCals(p, c, f int64) int64 {
	return c*4 + p*4 + f*9
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
		Calories: CalculateCals(cm.macros.Protein, cm.macros.Carbs, cm.macros.Fat),
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

type TodayMeal struct {
	out io.Writer
	db  *db.Queries
}

func NewTodayMeal(out io.Writer, db *db.Queries) *TodayMeal {
	return &TodayMeal{
		out: out,
		db:  db,
	}
}

func (tm *TodayMeal) TodayMeals() clic.HandlerFunc {
	return func(ctx context.Context) error {
		curDate := time.Now().In(time.Local).Format(time.DateOnly)

		logs, err := tm.db.ListMealLogsByDate(ctx, curDate)
		if err != nil {
			return err
		}

		if len(logs) == 0 {
			fmt.Fprintf(tm.out, "no meals logged for %s\n", curDate)
			return nil
		}

		fmt.Fprintf(tm.out, "Meals for %s:\n", curDate)
		for _, log := range logs {
			fmt.Fprintf(tm.out, "  %s - P: %d, C: %d, F: %d, Cals: %d\n", log.Name, log.Protein, log.Carbs, log.Fat, log.Calories)
		}

		totals, err := tm.db.GetDailyTotals(ctx, curDate)
		if err != nil {
			return err
		}

		fmt.Fprintf(tm.out, "--\nTotals: Protein: %v, Carbs: %v, Fat: %v, Cals: %v\n", totals.TotalProtein, totals.TotalCarbs, totals.TotalFat, totals.TotalCalories)
		return nil
	}
}

func NewMeal(out io.Writer, db *db.Queries) *Meal {
	return &Meal{
		out: out,
	}
}
func (m *Meal) HandleCommand(ctx context.Context) error {
	return nil
}
