package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/daved/clic"
	"github.com/dukedelaet/diet-tracker/internal/db"
)

//go:embed templates/*
var templateFS embed.FS

var (
	baseTmpl       = template.Must(template.ParseFS(templateFS, "templates/base.html"))
	weightScript   = template.Must(template.ParseFS(templateFS, "templates/weight.js"))
	mealScript     = template.Must(template.ParseFS(templateFS, "templates/meal.js"))
	exerciseScript = template.Must(template.ParseFS(templateFS, "templates/exercise.js"))
)

type chartData struct {
	Title       string
	Legend      string
	ChartScript string
}

type scriptData struct {
	Labels   string
	Data     string
	Tooltips string
}

type chart struct {
	out io.Writer
}

func NewChart(out io.Writer) *chart {
	return &chart{out: out}
}

func (c *chart) HandleCommand(ctx context.Context) error {
	fmt.Fprintln(c.out, "Usage: diet chart [weight|meal|exercise]")
	return nil
}

const inputDateFormat = "01-02-2006" // MM-DD-YYYY

// Helper to parse the start date, assuming the American way
// 06/22/1999
func parseStartDate(input string) (string, error) {
	if input == "" {
		return "0000-01-01", nil // Beginning of time
	}
	t, err := time.Parse(inputDateFormat, input)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use MM-DD-YYYY: %w", err)
	}
	return t.Format(time.DateOnly), nil
}

type chartWeight struct {
	out   io.Writer
	db    *db.Queries
	start string
}

func NewChartWeight(out io.Writer, db *db.Queries) *chartWeight {
	return &chartWeight{out: out, db: db}
}

func (cw *chartWeight) ChartWeight() clic.HandlerFunc {
	return func(ctx context.Context) error {
		startDate, err := parseStartDate(cw.start)
		if err != nil {
			return err
		}

		weights, err := cw.db.ListWeightsFromDate(ctx, startDate)
		if err != nil {
			return err
		}

		if len(weights) == 0 {
			fmt.Fprintln(cw.out, "No weight entries to chart")
			return nil
		}

		var labels, data []string
		for _, w := range weights {
			labels = append(labels, fmt.Sprintf(`"%s"`, w.Date))
			data = append(data, fmt.Sprintf("%d", w.Pounds))
		}

		script, err := renderScript(weightScript, scriptData{
			Labels: strings.Join(labels, ","),
			Data:   strings.Join(data, ","),
		})
		if err != nil {
			return err
		}

		return renderAndOpenChart(chartData{
			Title:       "Weight Progress (lbs)",
			ChartScript: script,
		}, "weight", cw.out)
	}
}

type chartMeal struct {
	out   io.Writer
	db    *db.Queries
	start string
}

func NewChartMeal(out io.Writer, db *db.Queries) *chartMeal {
	return &chartMeal{out: out, db: db}
}

func (cm *chartMeal) ChartMeal() clic.HandlerFunc {
	return func(ctx context.Context) error {
		startDate, err := parseStartDate(cm.start)
		if err != nil {
			return err
		}

		dailyCalories, err := cm.db.ListDailyCaloriesFromDate(ctx, startDate)
		if err != nil {
			return err
		}

		if len(dailyCalories) == 0 {
			fmt.Fprintln(cm.out, "No meal entries to chart")
			return nil
		}

		var labels, data []string
		for _, dc := range dailyCalories {
			labels = append(labels, fmt.Sprintf(`"%s"`, dc.Date))
			data = append(data, fmt.Sprintf("%v", dc.TotalCalories))
		}

		script, err := renderScript(mealScript, scriptData{
			Labels: strings.Join(labels, ","),
			Data:   strings.Join(data, ","),
		})
		if err != nil {
			return err
		}

		return renderAndOpenChart(chartData{
			Title:       "Daily Calorie Intake",
			ChartScript: script,
		}, "meal", cm.out)
	}
}

type chartExercise struct {
	out   io.Writer
	db    *db.Queries
	start string
}

func NewChartExercise(out io.Writer, db *db.Queries) *chartExercise {
	return &chartExercise{out: out, db: db}
}

func (ce *chartExercise) ChartExercise() clic.HandlerFunc {
	return func(ctx context.Context) error {
		startDate, err := parseStartDate(ce.start)
		if err != nil {
			return err
		}

		exercises, err := ce.db.ListExercisesFromDate(ctx, startDate)
		if err != nil {
			return err
		}

		if len(exercises) == 0 {
			fmt.Fprintln(ce.out, "No exercise entries to chart")
			return nil
		}

		// Group exercises by date for tooltip info
		type dayExercise struct {
			date      string
			exercises []db.Exercise
		}
		dateMap := make(map[string]*dayExercise)
		var orderedDates []string

		for _, ex := range exercises {
			if _, exists := dateMap[ex.Date]; !exists {
				dateMap[ex.Date] = &dayExercise{date: ex.Date}
				orderedDates = append(orderedDates, ex.Date)
			}
			dateMap[ex.Date].exercises = append(dateMap[ex.Date].exercises, ex)
		}

		var labels, data, tooltips []string
		for _, date := range orderedDates {
			de := dateMap[date]
			labels = append(labels, fmt.Sprintf(`"%s"`, de.date))
			data = append(data, "1") // 1 = exercised

			// Build tooltip with exercise details
			var details []string
			for _, ex := range de.exercises {
				details = append(details, fmt.Sprintf("%s: %d min", ex.ExerciseType, ex.Duration))
			}
			tooltips = append(tooltips, fmt.Sprintf(`"%s"`, strings.Join(details, ", ")))
		}

		script, err := renderScript(exerciseScript, scriptData{
			Labels:   strings.Join(labels, ","),
			Data:     strings.Join(data, ","),
			Tooltips: strings.Join(tooltips, ","),
		})
		if err != nil {
			return err
		}

		return renderAndOpenChart(chartData{
			Title:       "Exercise Activity",
			Legend:      "Hover over points to see exercise details",
			ChartScript: script,
		}, "exercise", ce.out)
	}
}

func renderScript(tmpl *template.Template, data scriptData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render script: %w", err)
	}
	return buf.String(), nil
}

func renderAndOpenChart(data chartData, chartType string, out io.Writer) error {
	var buf bytes.Buffer
	if err := baseTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render chart: %w", err)
	}

	filename := filepath.Join(os.TempDir(), fmt.Sprintf("diet-chart-%s-%d.html", chartType, time.Now().Unix()))
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write chart file: %w", err)
	}

	fmt.Fprintf(out, "Chart saved to %s\n", filename)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filename)
	case "linux":
		cmd = exec.Command("xdg-open", filename)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", filename)
	default:
		fmt.Fprintln(out, "Cannot auto-open chart on this platform. Please open the file manually.")
		return nil
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open chart: %w", err)
	}

	return nil
}
