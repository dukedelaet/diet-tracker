package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/daved/clic"
	"github.com/dukedelaet/diet-tracker/internal/db"
	"github.com/dukedelaet/diet-tracker/migrations"
	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

func main() {
	sqlDB, err := dbConn()
	if err != nil {
		log.Fatalf("db connection failed: %s", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("goose dialect: %s", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		log.Fatalf("goose up: %s", err)
	}

	w := os.Stdout
	q := db.New(sqlDB)

	// Weight commands
	listWeight := NewListWeight(w, q)
	listWeightClic := clic.NewFromFunc(listWeight.ListWeights(), "list")
	listWeightClic.Flag(&listWeight.limit, "l|limit", "Number of weight entries to show")

	weight := NewWeight(w, q)
	weightClic := clic.New(weight, "weight", listWeightClic)
	weightClic.Operand(&weight.pounds, false, "Current weight: Ex. 200", "Enter your weight in pounds for today")

	// Meal commands
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
	logMealClic.Flag(&logMeal.portion, "p|portion", "The portion of the meal eaten. Ex. 50 for half meal.")
	logMealClic.Operand(&logMeal.name, true, "Meal Name", "The meal must have already been created in order to log")

	todayMeal := NewTodayMeal(w, q)
	todayMealClic := clic.NewFromFunc(todayMeal.TodayMeals(), "today")

	meal := NewMeal(w, q)
	mealClic := clic.New(meal, "meal", createMealClic, listMealClic, logMealClic, todayMealClic)

	// Exercise commands
	createExercise := NewCreateExercise(w, q)
	createExerciseClic := clic.New(createExercise, "create")
	createExerciseClic.Flag(&createExercise.exerciseType, "t|type", "Exercise type (cardio or strength)")
	createExerciseClic.Flag(&createExercise.duration, "d|duration", "Duration in minutes")

	listExercise := NewListExercise(w, q)
	listExerciseClic := clic.NewFromFunc(listExercise.ListExercises(), "list")
	listExerciseClic.Flag(&listExercise.limit, "l|limit", "Number of exercises to show")

	exercise := NewExercise(w)
	exerciseClic := clic.New(exercise, "exercise", createExerciseClic, listExerciseClic)

	// Chart commands
	chartWeight := NewChartWeight(w, q)
	chartWeightClic := clic.NewFromFunc(chartWeight.ChartWeight(), "weight")
	chartWeightClic.Flag(&chartWeight.start, "s|start", "Start date (MM-DD-YYYY)")

	chartMeal := NewChartMeal(w, q)
	chartMealClic := clic.NewFromFunc(chartMeal.ChartMeal(), "meal")
	chartMealClic.Flag(&chartMeal.start, "s|start", "Start date (MM-DD-YYYY)")

	chartExercise := NewChartExercise(w, q)
	chartExerciseClic := clic.NewFromFunc(chartExercise.ChartExercise(), "exercise")
	chartExerciseClic.Flag(&chartExercise.start, "s|start", "Start date (MM-DD-YYYY)")

	chartHandler := NewChart(w)
	chartClic := clic.New(chartHandler, "chart", chartWeightClic, chartMealClic, chartExerciseClic)

	dietHandler := NewDietRoot(w)
	root := clic.New(dietHandler, "diet", weightClic, mealClic, exerciseClic, chartClic)

	// user error
	cmd, err := root.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprint(w, cmd.Usage())
		os.Exit(1)
	}

	// my error or expected error
	if err := cmd.Handle(context.Background()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Fprintln(w, "No entries in db")
		} else {
			log.Fatalln("keegan's bad", err)
		}
	}

}

func dbConn() (*sql.DB, error) {
	dsn, err := dbDSN()
	if err != nil {
		return nil, fmt.Errorf("dbConn: %w", err)
	}
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

func dbDSN() (string, error) {
	if p := strings.TrimSpace(os.Getenv("DIET_DB_PATH")); p != "" {
		err := os.MkdirAll(filepath.Dir(p), 0o755)
		if err != nil {
			return "", err
		}
		return "file:" + p, nil
	}

	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		p := filepath.Join(xdg, "diet-tracker", "app.db")
		err := os.MkdirAll(filepath.Dir(p), 0o755)
		if err != nil {
			return "", err
		}
		return "file:" + p, nil
	}

	home, _ := os.UserHomeDir()

	var p string
	if runtime.GOOS == "darwin" {
		p = filepath.Join(home, "Library", "Application Support", "diet-tracker", "app.db")
	} else {
		p = filepath.Join(home, ".local", "share", "diet-tracker", "app.db")
	}

	err := os.MkdirAll(filepath.Dir(p), 0o755)
	if err != nil {
		return "", err
	}
	return "file:" + p, nil
}
