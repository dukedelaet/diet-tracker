package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dukedelaet/diet-tracker/internal/db"
)

type rowKind int

const (
	kindWeight rowKind = iota
	kindMealLog
	kindExercise
	kindTotal
)

type row struct {
	kind rowKind
	id   int64
	text string
	meal *db.ListMealLogsByDateRow
}

func (r row) title() string        { return r.text }
func (r row) description() string { return "" }
func (r row) FilterValue() string { return "" }

type formScreen string

const (
	formWeight formScreen = "weight"
	formMeal   formScreen = "meal"
	formLog    formScreen = "log"
)

type model struct {
	width  int
	height int

	screen     string
	list       list.Model
	status     string
	err        string
	formActive bool
	form       []textinput.Model
	formIdx    int
	formScreen formScreen

	out   io.Writer
	db    *db.Queries
	ctx   context.Context
}

type NewModelOpts struct {
	DB     *sql.DB
	Out    io.Writer
	Ctx    context.Context
	Width  int
	Height int
}

func NewModel(opts NewModelOpts) *model {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	if opts.Width == 0 {
		opts.Width = 80
	}
	if opts.Height == 0 {
		opts.Height = 24
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	m := &model{
		width:  opts.Width,
		height: opts.Height,
		out:    opts.Out,
		db:     db.New(opts.DB),
		ctx:    opts.Ctx,
	}
	m.setupList()
	m.setupForm(formWeight)
	m.setupForm(formMeal)
	m.setupForm(formLog)
	return m
}

var forms = map[formScreen][]textinput.Model{}

func (m *model) setupList() {
	m.list = list.New(nil, rowDelegate{}, m.width, m.height)
	m.list.Title = "Today"
}

func (m *model) setupForm(s formScreen) {
	if _, ok := forms[s]; ok {
		return
	}
	switch s {
	case formWeight:
		lbs := textinput.New()
		lbs.Prompt = "lbs: "
		lbs.CharLimit = 5
		forms[s] = []textinput.Model{lbs}
	case formMeal:
		name := textinput.New()
		name.Prompt = "name: "
		p := textinput.New()
		p.Prompt = "p: "
		p.CharLimit = 4
		c := textinput.New()
		c.Prompt = "c: "
		c.CharLimit = 4
		f := textinput.New()
		f.Prompt = "f: "
		f.CharLimit = 4
		forms[s] = []textinput.Model{name, p, c, f}
	case formLog:
		t := textinput.New()
		t.Prompt = "type: "
		t.SetValue("cardio")
		d := textinput.New()
		d.Prompt = "min: "
		d.CharLimit = 4
		d.SetValue("30")
		forms[s] = []textinput.Model{t, d}
	}
}

func (m *model) Init() tea.Cmd {
	return m.refreshList()
}

func (m *model) refreshList() tea.Cmd {
	switch m.screen {
	case "today":
		return m.loadToday()
	case "weight":
		return m.loadRecentWeights()
	case "meals":
		return m.loadMeals()
	case "log":
		return m.loadLog()
	}
	return nil
}

func (m *model) loadToday() tea.Cmd {
	q, ctx, d := m.db, m.ctx, todayStr()
	return func() tea.Msg {
		var items []list.Item
		w, err := q.GetWeightByDate(ctx, d)
		switch {
		case err == nil:
			items = append(items, row{kind: kindWeight, id: w.ID, text: fmt.Sprintf("Weight %d lbs", w.Pounds)})
		case !errors.Is(err, sql.ErrNoRows):
			return errMsg{err}
		}
		logs, err := q.ListMealLogsByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		for _, l := range logs {
			l := l
			items = append(items, row{kind: kindMealLog, id: l.ID, meal: &l,
				text: fmt.Sprintf("%-24s  %dP %dC %dF  %dcals", l.Name, l.Protein, l.Carbs, l.Fat, l.Calories)})
		}
		exs, err := q.ListExercisesByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		for _, e := range exs {
			items = append(items, row{kind: kindExercise, id: e.ID,
				text: fmt.Sprintf("%s %d min", e.ExerciseType, e.Duration)})
		}
		tot, err := q.GetDailyTotals(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		items = append(items, row{kind: kindTotal,
			text: fmt.Sprintf("Totals  %dP %dC %dF  %dcals", float64AsInt(tot.TotalProtein), float64AsInt(tot.TotalCarbs), float64AsInt(tot.TotalFat), float64AsInt(tot.TotalCalories))})

		m.list.Title = "Today " + d
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadRecentWeights() tea.Cmd {
	q, ctx := m.db, m.ctx
	return func() tea.Msg {
		weights, err := q.ListWeights(ctx, 30)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		for _, w := range weights {
			items = append(items, row{kind: kindWeight, id: w.ID, text: fmt.Sprintf("%s  %d lbs", w.Date, w.Pounds)})
		}
		m.list.Title = "Recent weights"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadMeals() tea.Cmd {
	q, ctx := m.db, m.ctx
	d := todayStr()
	return func() tea.Msg {
		logs, err := q.ListMealLogsByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		for _, l := range logs {
			l := l
			items = append(items, row{kind: kindMealLog, id: l.ID, meal: &l,
				text: fmt.Sprintf("%-24s  %dP %dC %dF  %dcals", l.Name, l.Protein, l.Carbs, l.Fat, l.Calories)})
		}
		m.list.Title = "Meals logged today"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadLog() tea.Cmd {
	q, ctx, d := m.db, m.ctx, todayStr()
	return func() tea.Msg {
		exs, err := q.ListExercisesByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		for _, e := range exs {
			items = append(items, row{kind: kindExercise, id: e.ID,
				text: fmt.Sprintf("%s %d min", e.ExerciseType, e.Duration)})
		}
		m.list.Title = "Exercises today"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.formActive {
			return m.handleFormKey(msg)
		}
		return m.handleListKey(msg)
	}
	l, cmd := m.list.Update(msg)
	m.list = l
	return m, cmd
}

func (m *model) handleListKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, newKey):
		return m.beginForm()
	case key.Matches(k, delKey):
		if cur, ok := m.list.SelectedItem().(row); ok {
			return m.deleteRow(cur)
		}
		if cur, ok := m.list.SelectedItem().(row); ok && cur.meal != nil {
			// Just show a status; actual DB write would need a separate action.
			// For now this is a placeholder for the half-portion feature.
			_ = cur
		}
	case key.Matches(k, todayKey):
		m.screen = "today"
		return m, m.refreshList()
	case key.Matches(k, weightKey):
		m.screen = "weight"
		return m, m.refreshList()
	case key.Matches(k, mealsKey):
		m.screen = "meals"
		return m, m.refreshList()
	case key.Matches(k, logKey):
		m.screen = "log"
		return m, m.refreshList()
	case key.Matches(k, quitKey):
		return m, tea.Quit
	}
	l, cmd := m.list.Update(k)
	m.list = l
	return m, cmd
}

func (m *model) beginForm() (tea.Model, tea.Cmd) {
	switch m.screen {
	case "weight":
		m.formScreen = formWeight
	case "meals":
		m.formScreen = formMeal
	case "log":
		m.formScreen = formLog
	}
	m.form = forms[m.formScreen]
	m.formActive = true
	m.formIdx = 0
	m.err = ""
	return m, m.form[0].Focus()
}

func (m *model) handleFormKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case k.Type == tea.KeyCtrlC:
		return m, tea.Quit
	case k.Type == tea.KeyEsc:
		m.formActive = false
		m.formIdx = 0
		return m, m.refreshList()
	case k.Type == tea.KeyEnter && m.formIdx == len(m.form)-1:
		return m.submitForm()
	case k.Type == tea.KeyTab:
		m.formIdx = (m.formIdx + 1) % len(m.form)
		return m, m.form[m.formIdx].Focus()
	case k.Type == tea.KeyShiftTab:
		m.formIdx = (m.formIdx - 1 + len(m.form)) % len(m.form)
		return m, m.form[m.formIdx].Focus()
	}
	ti, cmd := m.form[m.formIdx].Update(k)
	m.form[m.formIdx] = ti
	return m, cmd
}

func intOr(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	return strconv.Atoi(s)
}

func (m *model) submitForm() (tea.Model, tea.Cmd) {
	f := m.form
	switch m.formScreen {
	case formWeight:
		lbs, err := intOr(f[0].Value(), 0)
		if err != nil || lbs <= 0 {
			m.err = "invalid pounds"
			return m, nil
		}
		return m, func() tea.Msg {
			d := todayStr()
			if err := m.db.UpdateWeight(m.ctx, db.UpdateWeightParams{Pounds: int64(lbs), Date: d}); err == nil {
				m.status = fmt.Sprintf("updated weight to %d lbs", lbs)
			} else if errors.Is(err, sql.ErrNoRows) {
				if _, err := m.db.CreateWeight(m.ctx, db.CreateWeightParams{Pounds: int64(lbs), Date: d}); err != nil {
					return errMsg{err}
				}
				m.status = fmt.Sprintf("logged weight %d lbs", lbs)
			} else {
				return errMsg{err}
			}
			m.form = forms[formWeight]
			m.formActive = false
			return m.refreshList()
		}
	case formMeal:
		name := f[0].Value()
		p, e1 := intOr(f[1].Value(), 0)
		c, e2 := intOr(f[2].Value(), 0)
		fat, e3 := intOr(f[3].Value(), 0)
		if name == "" || e1 != nil || e2 != nil || e3 != nil {
			m.err = "fill name and all three macros"
			return m, nil
		}
		if p+c+fat == 0 {
			m.err = "macros cannot all be zero"
			return m, nil
		}
		cals := c*4 + p*4 + fat*9
		return m, func() tea.Msg {
			if _, err := m.db.GetMealByName(m.ctx, name); err == nil {
				return errMsg{errors.New("meal already exists")}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return errMsg{err}
			}
			if _, err := m.db.CreateMeal(m.ctx, db.CreateMealParams{Name: name, Protein: int64(p), Carbs: int64(c), Fat: int64(fat), Calories: int64(cals)}); err != nil {
				return errMsg{err}
			}
			m.status = fmt.Sprintf("created meal %s (%d cals)", name, cals)
			m.form = forms[formMeal]
			for _, x := range m.form {
				x.SetValue("")
			}
			m.formActive = false
			return m.refreshList()
		}
	case formLog:
		t := strings.ToLower(strings.TrimSpace(f[0].Value()))
		dur, err := intOr(f[1].Value(), 0)
		if err != nil || (t != "cardio" && t != "strength") || dur <= 0 {
			m.err = "type must be cardio/strength, minutes > 0"
			return m, nil
		}
		return m, func() tea.Msg {
			if _, err := m.db.CreateExercise(m.ctx, db.CreateExerciseParams{Date: todayStr(), ExerciseType: t, Duration: int64(dur)}); err != nil {
				return errMsg{err}
			}
			m.status = fmt.Sprintf("logged %d min %s", dur, t)
			f[1].SetValue("30")
			f[0].SetValue("cardio")
			m.formActive = false
			return m.refreshList()
		}
	}
	return m, nil
}

func (m *model) deleteRow(r row) (tea.Model, tea.Cmd) {
	id := r.id
	deleteFn := func() error {
		switch r.kind {
		case kindWeight:
			return m.db.DeleteWeight(m.ctx, id)
		case kindMealLog:
			return m.db.DeleteMealLog(m.ctx, id)
		case kindExercise:
			return m.db.DeleteExercise(m.ctx, id)
		default:
			return nil
		}
	}
	return m, func() tea.Msg {
		if err := deleteFn(); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return errMsg{err}
		}
		m.status = "deleted entry"
		return m.refreshList()
	}
}

func (m *model) View() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "\n  1) today  2) weight  3) meals  4) log\n\n")
	fmt.Fprint(&sb, m.list.View())
	if m.formActive {
		fmt.Fprint(&sb, "\n\n  FORM — tab to move, esc cancel, enter submit\n")
		for i, f := range m.form {
			prompt := f.Prompt
			if i != m.formIdx {
				prompt = "  "
			}
			fmt.Fprintf(&sb, "  %-8s %s\n", prompt, f.View())
		}
	} else {
		fmt.Fprint(&sb, "\n  n: new  x: delete  q: quit\n")
	}
	if m.err != "" {
		fmt.Fprintf(&sb, "\n  ⚠  %s\n", m.err)
	}
	if m.status != "" {
		fmt.Fprintf(&sb, "\n  ✓  %s\n", m.status)
	}
	return sb.String()
}

func (m *model) Run() {
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil && m.out != nil {
		panic(err)
	}
}

var (
	todayKey   = key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "today"))
	weightKey  = key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "weight"))
	mealsKey   = key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "meals"))
	logKey     = key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "today activity"))
	quitKey    = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	newKey     = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new entry"))
	delKey     = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete row"))
)

func todayStr() string {
	return time.Now().In(time.Local).Format(time.DateOnly)
}

func float64AsInt(v interface{}) int {
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return int(f)
}

type errMsg struct{ err error }
type msg struct{}

type rowDelegate struct{ list.DefaultDelegate }

func (rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(row)
	if !ok {
		return
	}
	fmt.Fprint(w, r.text)
}
