# Diet Tracker

This is a cli app to make tracking you diet easy and terminal friendly!

## Usage

### Installation

```bash
go install ./cmd/diet
```

### Database Location

The database is stored automatically at:
- **macOS**: `~/Library/Application Support/diet-tracker/app.db`
- **Linux**: `~/.local/share/diet-tracker/app.db`

Override with the `DIET_DB_PATH` environment variable if needed.

### Commands

#### Weight

```bash
diet weight 175              # Log today's weight (lbs)
diet weight list             # Show last 7 entries
diet weight list -l 30       # Show last 30 entries
```

#### Meals

```bash
# Create a meal template
diet meal create -n "Chicken Rice" -p 40 -c 50 -f 10

# List saved meals
diet meal list

# Log a meal
diet meal log "Chicken Rice"
diet meal log "Chicken Rice" -p 50   # Log 50% portion

# View today's intake with totals
diet meal today
```

#### Exercise

```bash
# Log exercise (type: cardio or strength)
diet exercise create -t cardio -d 30      # 30 min cardio
diet exercise create -t strength -d 45    # 45 min strength

# List recent exercises
diet exercise list
```

#### Charts

Generate interactive charts that open in your browser:

```bash
diet chart weight                    # Weight over time
diet chart meal                      # Daily calorie intake
diet chart exercise                  # Exercise activity
diet chart weight -s 01-01-2026      # Start from specific date (MM-DD-YYYY)
```

### Example Workflow

```bash
# Set up your common meals once
diet meal create -n "Oatmeal Breakfast" -p 31 -c 56 -f 5
diet meal create -n "Grilled Chicken Salad" -p 45 -c 15 -f 12

# Daily tracking
diet weight 175
diet meal log "Oatmeal Breakfast"
diet meal log "Grilled Chicken Salad"
diet exercise create -t cardio -d 30

# Check progress
diet meal today
diet chart weight
```

## Contributing

I am doing this guy solo right now, but please fork it and add
features to your heart's content.
