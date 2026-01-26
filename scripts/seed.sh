#!/bin/bash

# Seed regular meals into the database

# Meal 1: Oats + Legion Protein + Banana
# 1/2 cup oats: 150 cal, 27c, 5p, 3f
# 1 cup Legion protein: 120 cal, 2c, 25p, 1.5f
# Medium banana: 105 cal, 27c, 1p, 0.5f
# Total: 375 cal, 56c, 31p, 5f
diet meal create -n "Oats Protein Banana" -p 31 -c 56 -f 5

# Meal 2: Spaghetti with Marinara, Ground Beef, Broccoli
# 1.5 cups cooked spaghetti: 330 cal, 65c, 12p, 2f
# Marinara (1/2 cup): 60 cal, 12c, 2p, 1f
# Ground beef 4oz (85/15): 240 cal, 0c, 24p, 15f
# Roasted broccoli (1 cup): 55 cal, 11c, 4p, 0.5f
# Total: 685 cal, 88c, 42p, 18f
diet meal create -n "Spaghetti Ground Beef" -p 42 -c 88 -f 18

# Meal 3: Salmon with White Rice and Veggies
# Atlantic salmon (6oz): 360 cal, 0c, 40p, 21f
# 1.5 cups cooked white rice: 300 cal, 67c, 6p, 0.5f
# Roasted veggies (1 cup): 60 cal, 12c, 2p, 0.5f
# Total: 720 cal, 79c, 48p, 22f
diet meal create -n "Salmon Rice Veggies" -p 48 -c 79 -f 22

# Meal 4: Factor Meal (Low Protein)
# Estimate: 500 cal, 40c, 30p, 20f
diet meal create -n "Factor Low Protein" -p 30 -c 40 -f 20

# Meal 5: Factor Meal (High Protein)
# Estimate: 600 cal, 40c, 40p, 25f
diet meal create -n "Factor High Protein" -p 40 -c 40 -f 25

echo "All meals seeded successfully!"
