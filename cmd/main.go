package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/daved/clic"
	"github.com/guitarkeegan/diet-tracker/internal/db"

	_ "modernc.org/sqlite"
)

func main() {
	sqlDB, err := dbConn("file:./app.db")
	if err != nil {
		log.Fatalf("db connection failed: %s", err)
	}
	defer sqlDB.Close()

	w := os.Stdout
	q := db.New(sqlDB)

	weight := NewWeight(w, q)
	weightClic := clic.New(weight, "weight")
	weightClic.Operand(&weight.pounds, false, "Log Weight", "Enter your weight in pounds for today")

	createMeal := NewCreateMeal(w, q)
	createMealClic := clic.New(createMeal, "create")
	createMealClic.Flag(&createMeal.name, "n|name", "Name of the meal")
	createMealClic.Flag(&createMeal.macros.Protein, "p|protein", "Grams of protein in meal")
	createMealClic.Flag(&createMeal.macros.Carbs, "c|carbs", "Grams of carbs in meal")
	createMealClic.Flag(&createMeal.macros.Fat, "f|fat", "Grams of fat in meal")

	listMeal := NewListMeal(w, q)
	listMealClic := clic.NewFromFunc(listMeal.ListMeals(), "list")
	listMealClic.Flag(&listMeal.limit, "l|limit", "Set max meals to return")

	logMeal := NewLogMeal(w, q)
	logMealClic := clic.NewFromFunc(logMeal.LogMeal(), "log")
	logMealClic.Operand(&logMeal.name, true, "Meal Name", "The meal must have already been created in order to log")

	meal := NewMeal(w, q)
	mealClic := clic.New(meal, "meal", createMealClic, listMealClic, logMealClic)

	dietHandler := NewDietRoot(w)
	root := clic.New(dietHandler, "diet", weightClic, mealClic)

	cmd, err := root.Parse(os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}

	if err := cmd.Handle(context.Background()); err != nil {
		log.Fatalln(err)
		fmt.Fprint(w, root.Usage())
	}

}

func dbConn(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -64000;
		PRAGMA foreign_keys = ON;
	`

	if _, err := db.Exec(pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set PRAGMAs: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
