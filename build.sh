#!/bin/bash

# Build the application
echo "Building OpenPrompt..."
go build -o openprompt ./cmd

if [ $? -eq 0 ]; then
    echo "Build successful! The binary is located at: ./openprompt"
else
    echo "Build failed."
    exit 1
fi
