package ui

import (
	"fmt"
	"os"
	"time"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	
	"github.com/openprompt/internal/fileutils"
	"github.com/openprompt/internal/preferences"
	"github.com/openprompt/internal/prompt"
)

type App struct {
	fyneApp    fyne.App
	window     fyne.Window
	fileTree   *FileTreeWidget
	currentDir string
	
	// Prompt controls
	instructionsEntry *widget.Entry
	
	// Token estimation
	tokenEstimation *widget.Label
	
	// Generated XML prompt
	xmlPrompt string
	
	// Preferences
	prefs *preferences.Preferences
}

func NewApp() *App {
	return &App{
		fyneApp: app.New(),
	}
}

func (a *App) Run() {
	a.window = a.fyneApp.NewWindow("OpenPrompt - LLM File Prompt Generator")
	a.window.Resize(fyne.NewSize(1200, 800)) // Larger window size
	
	// Load preferences
	var err error
	a.prefs, err = preferences.Load()
	if err != nil {
		dialog.ShowError(err, a.window)
	}
	
	// Create file tree
	fileTree := a.createFileTree()
	
	// Create prompt controls
	promptControls := a.createPromptControls()
	
	// Create token estimation and clipboard controls
	actionControls := a.createActionControls()
	
	// Create directory selection and refresh buttons
	dirButton := widget.NewButton("Select Directory", func() {
		a.selectDirectory()
	})
	
	refreshButton := widget.NewButton("Refresh File Tree", func() {
		if a.currentDir == "" {
			dialog.ShowInformation("Error", "Please select a directory first", a.window)
			return
		}
		
		// Force rebuild the file tree
		fmt.Println("Manually refreshing file tree for:", a.currentDir)
		a.applyFilters()
	})
	
	// Layout the UI
	dirButtons := container.NewHBox(
		dirButton,
		refreshButton,
	)
	
	rightPanel := container.NewVBox(
		dirButtons,
		promptControls,
		actionControls,
	)
	
	// Create a container that will fill the available space
	fileTreeContainer := container.NewMax(fileTree)
	
	// Add a border to make the file tree more visible
	fileTreeWithBorder := container.NewBorder(nil, nil, nil, nil, fileTreeContainer)
	
	content := container.NewHSplit(
		fileTreeWithBorder,
		rightPanel,
	)
	content.SetOffset(0.3) // 30% for file tree, 70% for controls
	
	a.window.SetContent(content)
	
	// Load last directory if available
	lastDir := a.prefs.GetLastDirectory()
	if lastDir != "" {
		// Check if directory exists
		if info, err := os.Stat(lastDir); err == nil && info.IsDir() {
			fmt.Println("Loading last directory:", lastDir)
			a.currentDir = lastDir
			
			// Apply filters with a slight delay to ensure the UI is fully initialized
			go func() {
				// Small delay to ensure UI is ready
				time.Sleep(200 * time.Millisecond)
				
				// Apply filters in the main thread
				a.window.Canvas().Refresh(a.fileTree)
				a.applyFilters()
			}()
		}
	}
	
	// Set up window close event to save preferences
	a.window.SetOnClosed(func() {
		if a.prefs != nil {
			a.prefs.Save()
		}
	})
	
	a.window.ShowAndRun()
}

func (a *App) createFileTree() fyne.CanvasObject {
	a.fileTree = NewFileTreeWidget(func() {
		// When selection changes, update token estimation
		a.updateTokenEstimation()
	})
	
	return container.NewVBox(
		widget.NewLabel("File Tree (select files to include)"),
		container.NewBorder(nil, nil, nil, nil, a.fileTree),
	)
}

func (a *App) createPromptControls() fyne.CanvasObject {
	// Instructions for the LLM
	instructionsLabel := widget.NewLabel("Instructions for LLM:")
	a.instructionsEntry = widget.NewEntry()
	a.instructionsEntry.SetPlaceHolder("Analyze the code and suggest improvements...")
	a.instructionsEntry.MultiLine = true
	a.instructionsEntry.Wrapping = fyne.TextWrapWord
	a.instructionsEntry.SetMinRowsVisible(5)
	a.instructionsEntry.OnChanged = func(text string) {
		a.updateTokenEstimation()
	}
	
	return container.NewVBox(
		widget.NewCard("LLM Instructions", "", container.NewVBox(
			instructionsLabel,
			a.instructionsEntry,
		)),
	)
}

func (a *App) createActionControls() fyne.CanvasObject {
	// Token estimation
	a.tokenEstimation = widget.NewLabel("Estimated Tokens: 0")
	
	// Model token limit
	limitLabel := widget.NewLabel("Model Token Limit:")
	limitEntry := widget.NewEntry()
	limitEntry.SetText("8192") // Default for GPT-4
	
	// Generate and copy button
	generateButton := widget.NewButton("Generate & Copy to Clipboard", func() {
		a.generateAndCopy()
	})
	
	// Preview XML button
	previewButton := widget.NewButton("Preview XML", func() {
		a.previewXML()
	})
	
	return container.NewVBox(
		widget.NewCard("Actions", "", container.NewVBox(
			a.tokenEstimation,
			container.NewHBox(limitLabel, limitEntry),
			generateButton,
			previewButton,
		)),
	)
}

func (a *App) selectDirectory() {
	// Create a dialog with both browse and manual entry options
	content := container.NewVBox()
	
	// Manual path entry
	pathLabel := widget.NewLabel("Enter Directory Path:")
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("/path/to/directory")
	
	// Set current directory if available
	if a.currentDir != "" {
		pathEntry.SetText(a.currentDir)
	}
	
	// Create the dialog first so we can reference it
	dirDialog := dialog.NewCustom("Select Directory", "Cancel", content, a.window)
	
	// Buttons
	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			
			pathEntry.SetText(uri.Path())
		}, a.window)
	})
	
	confirmButton := widget.NewButton("Confirm", func() {
		path := pathEntry.Text
		if path == "" {
			dialog.ShowInformation("Error", "Please enter a directory path", a.window)
			return
		}
		
		// Check if directory exists
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			dialog.ShowInformation("Error", "Invalid directory path", a.window)
			return
		}
		
		// Close the dialog first
		dirDialog.Hide()
		
		// Set current directory
		a.currentDir = path
		
		// Save to preferences
		if a.prefs != nil {
			a.prefs.SetLastDirectory(path)
			a.prefs.Save()
		}
		
		// Apply filters with a slight delay to ensure the dialog is fully closed
		go func() {
			// Small delay to ensure dialog is closed
			time.Sleep(100 * time.Millisecond)
			
			fmt.Println("Loading directory:", path)
			a.applyFilters()
		}()
	})
	
	// Layout
	content.Add(pathLabel)
	content.Add(pathEntry)
	content.Add(container.NewHBox(
		browseButton,
		widget.NewLabel(""), // Spacer
		confirmButton,
	))
	
	// Show dialog
	dirDialog.Show()
}

func (a *App) applyFilters() {
	if a.currentDir == "" {
		dialog.ShowInformation("Error", "Please select a directory first", a.window)
		return
	}
	
	fmt.Println("Loading directory:", a.currentDir)
	
	// Use default filters
	filters := fileutils.FileFilters{
		// No filters - show all files
		RespectGitignore: true, // Respect .gitignore files by default
	}
	
	// Load files
	err := a.fileTree.LoadDirectory(a.currentDir, filters)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	
	// Update token estimation
	a.updateTokenEstimation()
}

func (a *App) updateTokenEstimation() {
	// Get selected files
	selectedFiles := a.fileTree.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		a.tokenEstimation.SetText("Estimated Tokens: 0")
		return
	}
	
	// Don't generate XML here, just estimate based on file sizes
	totalSize := 0
	for _, file := range selectedFiles {
		if !file.IsDir {
			// Estimate based on file size
			info, err := os.Stat(file.Path)
			if err == nil {
				totalSize += int(info.Size())
			}
		}
	}
	
	// Rough estimate: 1 token per 4 characters
	estimatedTokens := totalSize / 4
	
	// Format the token count
	formattedTokens := fileutils.FormatTokenCount(estimatedTokens)
	
	// Update token estimation label
	a.tokenEstimation.SetText(fmt.Sprintf("Estimated Tokens: ~%s (rough estimate)", formattedTokens))
	
	// Check if exceeds limit
	limit := 8192 // Default for GPT-4
	if estimatedTokens > limit {
		a.tokenEstimation.SetText(fmt.Sprintf("Estimated Tokens: ~%s (exceeds limit of %s)", 
			formattedTokens, fileutils.FormatTokenCount(limit)))
	}
}

func (a *App) generateAndCopy() {
	// Get selected files
	selectedFiles := a.fileTree.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		dialog.ShowInformation("Error", "No files selected. Please select files first.", a.window)
		return
	}
	
	// Create a progress dialog
	progress := dialog.NewProgress("Generating XML", "Processing files...", a.window)
	progress.Show()
	
	// Generate XML in a goroutine to keep UI responsive
	go func() {
		// Generate XML
		xmlPrompt, err := prompt.GenerateXML(selectedFiles, a.instructionsEntry.Text, a.currentDir)
		
		// Save XML for later use
		a.xmlPrompt = xmlPrompt
		
		// Complete the progress
		progress.SetValue(1.0)
		
		// Small delay to ensure progress bar shows completion
		time.Sleep(100 * time.Millisecond)
		
		// Hide the progress dialog
		progress.Hide()
		
		// Handle errors or continue
		if err != nil {
			// We need to use the main thread for dialog operations
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Error",
				Content: err.Error(),
			})
			return
		}
		
		// Copy to clipboard
		err = prompt.CopyToClipboard(a.xmlPrompt)
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Error",
				Content: "Failed to copy to clipboard: " + err.Error(),
			})
			return
		}
		
		// Show success notification
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Success",
			Content: "XML prompt copied to clipboard",
		})
	}()
}

func (a *App) previewXML() {
	// Get selected files
	selectedFiles := a.fileTree.GetSelectedFiles()
	if len(selectedFiles) == 0 {
		dialog.ShowInformation("Error", "No files selected. Please select files first.", a.window)
		return
	}
	
	// Create a progress dialog
	progress := dialog.NewProgress("Generating XML", "Processing files...", a.window)
	progress.Show()
	
	// Generate XML in a goroutine to keep UI responsive
	go func() {
		// Generate XML
		xmlPrompt, err := prompt.GenerateXML(selectedFiles, a.instructionsEntry.Text, a.currentDir)
		
		// Save XML for later use
		a.xmlPrompt = xmlPrompt
		
		// Complete the progress
		progress.SetValue(1.0)
		
		// Small delay to ensure progress bar shows completion
		time.Sleep(100 * time.Millisecond)
		
		// Hide the progress dialog
		progress.Hide()
		
		// Handle errors
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Error",
				Content: err.Error(),
			})
			return
		}
		
		// We need to return to the main thread for UI operations
		// For now, we'll just show a notification to check the console
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "XML Preview",
			Content: "XML preview is ready. Check the console output.",
		})
		
		// Print the XML to console for now
		fmt.Println("XML Preview:")
		fmt.Println(a.xmlPrompt)
	}()
}
