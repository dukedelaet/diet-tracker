package main

import (
	"github.com/dukedelaet/diet-tracker/internal/db"
	"io"
	"testing"
)

func TestCreateMeal_validateCreateMeal(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		out     io.Writer
		db      *db.Queries
		meal    CreateMeal
		wantErr error
	}{
		{
			name: "must have macros",
			out:  nil,
			db:   &db.Queries{},
			meal: CreateMeal{
				name: "steak",
				macros: Macros{
					Protein: 0,
					Carbs:   0,
					Fat:     0,
				},
			},
			wantErr: noMacrosErr,
		},
		{
			name: "must have name",
			out:  nil,
			db:   &db.Queries{},
			meal: CreateMeal{
				name: "",
				macros: Macros{
					Protein: 10,
					Carbs:   5,
					Fat:     3,
				},
			},
			wantErr: missingNameErr,
		},
		{
			name: "missing name and macros returns macros error",
			out:  nil,
			db:   &db.Queries{},
			meal: CreateMeal{
				name: "",
				macros: Macros{
					Protein: 0,
					Carbs:   0,
					Fat:     0,
				},
			},
			wantErr: noMacrosErr,
		},
		{
			name: "valid meal",
			out:  nil,
			db:   &db.Queries{},
			meal: CreateMeal{
				name: "oatmeal",
				macros: Macros{
					Protein: 5,
					Carbs:   27,
					Fat:     3,
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewCreateMeal(tt.out, tt.db)
			cm.name = tt.meal.name
			cm.macros = tt.meal.macros
			gotErr := cm.validateCreateMeal()
			if gotErr != tt.wantErr {
				t.Fatalf("validateCreateMeal() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}
