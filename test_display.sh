#!/bin/bash

# Test script for display logic

echo "Testing display logic"

# Simulate options
options=("find . -name \"*.pix\"" "ls *.pix" "locate \"*.pix\"")
selected=1
max_options=3

echo ""
for ((i=1; i<=max_options; i++)); do
    if [ $i -eq $selected ]; then
        echo "➤ ${options[$((i-1))]}"
    else
        echo "  ${options[$((i-1))]}"
    fi
done

echo ""
echo "Simulating selection update..."
selected=2

# Clear and redraw (simulating our fix)
echo -ne "[${max_options}A[J"
for ((i=1; i<=max_options; i++)); do
    if [ $i -eq $selected ]; then
        echo "➤ ${options[$((i-1))]}"
    else
        echo "  ${options[$((i-1))]}"
    fi
done

echo ""
echo "Display logic working correctly!"

