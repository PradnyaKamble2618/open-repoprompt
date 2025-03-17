package ui

import (
	"fmt"
	"path/filepath"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/openprompt/internal/fileutils"
)

// FileTreeWidget is a custom widget for displaying a file tree
type FileTreeWidget struct {
	widget.BaseWidget
	container    *fyne.Container
	files        []*fileutils.FileInfo
	onChanged    func()
	expandedDirs map[string]bool // Track expanded directories
	currentDir   string          // Current root directory
	filters      fileutils.FileFilters // Current filters
	checkboxes   map[string]*widget.Check // Track checkboxes for direct updates
}

// NewFileTreeWidget creates a new file tree widget
func NewFileTreeWidget(onChanged func()) *FileTreeWidget {
	t := &FileTreeWidget{
		onChanged:    onChanged,
		expandedDirs: make(map[string]bool),
		checkboxes:   make(map[string]*widget.Check),
	}
	t.ExtendBaseWidget(t)
	
	// Create an empty container
	t.container = container.NewVBox()
	
	return t
}

// CreateRenderer creates a renderer for the file tree widget
func (t *FileTreeWidget) CreateRenderer() fyne.WidgetRenderer {
	scroll := container.NewScroll(t.container)
	scroll.SetMinSize(fyne.NewSize(300, 700)) // Set larger minimum size
	
	// Create a border container to make the file tree more visible
	border := container.NewBorder(nil, nil, nil, nil, scroll)
	
	return widget.NewSimpleRenderer(border)
}

// MinSize returns the minimum size of the widget
func (t *FileTreeWidget) MinSize() fyne.Size {
	return fyne.NewSize(300, 700)
}

// LoadDirectory loads files from a directory
func (t *FileTreeWidget) LoadDirectory(dir string, filters fileutils.FileFilters) error {
	// Store current directory and filters for later use
	t.currentDir = dir
	t.filters = filters
	
	// List only the top-level files with the given filters
	files, err := fileutils.ListFiles(dir, filters)
	if err != nil {
		return err
	}
	
	// Build file tree but only for the top level
	t.files = fileutils.BuildFileTree(files)
	
	// Calculate token counts for directories
	for _, file := range t.files {
		if file.IsDir {
			fileutils.CalculateDirectoryTokenCount(file)
		}
	}
	
	// Debug output
	fmt.Printf("Loaded %d root files/directories\n", len(t.files))
	
	// Rebuild the UI directly to avoid nil pointer issues
	t.rebuildUI()
	
	return nil
}

// rebuildUI rebuilds the UI based on the current files
func (t *FileTreeWidget) rebuildUI() {
	// Clear the container
	t.container.RemoveAll()
	
	// Add all root files
	for _, file := range t.files {
		t.addFileToUI(file, 0)
	}
	
	// Refresh the widget
	t.Refresh()
}

// addFileToUI adds a file to the UI
func (t *FileTreeWidget) addFileToUI(file *fileutils.FileInfo, indent int) {
	// Create a checkbox for selection
	check := widget.NewCheck("", nil) // Initialize with nil to prevent recursive calls
	check.OnChanged = func(checked bool) {
		file.Selected = checked
		
		// If it's a directory, select/unselect all children
		if file.IsDir {
			// Use a goroutine for potentially expensive operations
			go func() {
				// If children aren't loaded yet and we're selecting the directory, load them now
				if checked && len(file.Children) == 0 {
					t.loadChildren(file)
				}
				
				// Select/unselect all children
				t.toggleSelection(file, checked)
				
				// Notify of change on the main thread
				fyne.CurrentApp().Driver().CanvasForObject(t).Content().Refresh()
				if t.onChanged != nil {
					t.onChanged()
				}
			}()
		} else {
			// For regular files, just notify of change
			if t.onChanged != nil {
				t.onChanged()
			}
		}
	}
	check.Checked = file.Selected
	
	// Store the checkbox for direct updates
	t.checkboxes[file.Path] = check
	
	// Create a label for the file name
	name := filepath.Base(file.Path)
	if name == "" {
		name = file.Path // Use full path if base name is empty
	}
	
	// Create indentation
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "    "
	}
	
	// Use different icon based on file type
	var icon fyne.Resource
	if file.IsDir {
		icon = theme.FolderIcon()
	} else {
		icon = theme.DocumentIcon()
	}
	
	// Create token count label with formatted count
	tokenLabel := widget.NewLabel(fmt.Sprintf("[%s tokens]", fileutils.FormatTokenCount(file.TokenCount)))
	
	// Create a container for the file/directory
	var item *fyne.Container
	
	if file.IsDir {
		// For directories, add an expand/collapse button
		expandButton := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
			t.toggleExpand(file)
		})
		
		// Set the button icon based on expanded state
		if t.expandedDirs[file.Path] {
			expandButton.SetIcon(theme.MoveDownIcon())
		} else {
			expandButton.SetIcon(theme.NavigateNextIcon())
		}
		
		// Create a container with checkbox, icon, label, token count, and expand button
		item = container.NewBorder(
			nil, nil, 
			container.NewHBox(
				widget.NewLabel(indentStr),
				check,
				widget.NewIcon(icon),
				widget.NewLabel(name),
				tokenLabel,
			),
			expandButton,
		)
	} else {
		// For files, just show checkbox, icon, label, and token count
		item = container.NewHBox(
			widget.NewLabel(indentStr),
			check,
			widget.NewIcon(icon),
			widget.NewLabel(name),
			tokenLabel,
		)
	}
	
	// Add the item to the container
	t.container.Add(item)
	
	// Add children if this is an expanded directory
	if file.IsDir && t.expandedDirs[file.Path] {
		// If children aren't loaded yet, load them now
		if len(file.Children) == 0 {
			t.loadChildren(file)
		}
		
		// Display children
		for _, child := range file.Children {
			t.addFileToUI(child, indent+1)
		}
	}
}

// loadChildren loads children for a directory
func (t *FileTreeWidget) loadChildren(dir *fileutils.FileInfo) {
	// Only load if it's a directory and has no children yet
	if !dir.IsDir {
		return
	}
	
	// Get the full path to the directory
	var dirPath string
	if filepath.IsAbs(dir.Path) {
		// If the path is absolute, use it directly
		dirPath = dir.Path
	} else {
		// Otherwise, join with the current directory
		dirPath = filepath.Join(t.currentDir, dir.Path)
	}
	
	// Create filters for this subdirectory
	subFilters := t.filters
	subFilters.SubPath = filepath.Base(dir.Path)
	
	// List files in this directory
	files, err := fileutils.ListFiles(filepath.Dir(dirPath), subFilters)
	if err != nil {
		fmt.Printf("Error loading children for %s: %v\n", dir.Path, err)
		return
	}
	
	// Build file tree for these files
	children := fileutils.BuildFileTree(files)
	
	// Set the children
	dir.Children = children
	
	// Calculate token counts for child directories
	for _, child := range dir.Children {
		if child.IsDir {
			fileutils.CalculateDirectoryTokenCount(child)
		}
	}
	
	// Update the token count for this directory
	fileutils.CalculateDirectoryTokenCount(dir)
	
	fmt.Printf("Loaded %d children for %s\n", len(children), dir.Path)
}

// toggleExpand toggles the expanded state of a directory
func (t *FileTreeWidget) toggleExpand(dir *fileutils.FileInfo) {
	if !dir.IsDir {
		return
	}
	
	// Toggle expanded state
	t.expandedDirs[dir.Path] = !t.expandedDirs[dir.Path]
	
	// If we're expanding and there are no children yet, load them
	if t.expandedDirs[dir.Path] && len(dir.Children) == 0 {
		t.loadChildren(dir)
	}
	
	// Rebuild UI - this is necessary when expanding/collapsing
	t.rebuildUI()
}

// toggleSelection toggles the selection state of a file and its children
func (t *FileTreeWidget) toggleSelection(file *fileutils.FileInfo, selected bool) {
	// Update the selection state
	file.Selected = selected
	
	// Update the checkbox directly
	if check, ok := t.checkboxes[file.Path]; ok {
		check.SetChecked(selected)
	}
	
	// If it's a directory, recursively update all children
	if file.IsDir {
		// If children aren't loaded yet and we're selecting, load them now
		if selected && len(file.Children) == 0 {
			// Use a goroutine to load children asynchronously
			go func() {
				t.loadChildren(file)
				// Update UI on the main thread after loading
				fyne.CurrentApp().Driver().CanvasForObject(t).Content().Refresh()
			}()
		}
		
		// Update all children (only for already loaded children)
		for _, child := range file.Children {
			// Set the selected state directly without recursion
			child.Selected = selected
			
			// Update the checkbox directly if it exists
			if check, ok := t.checkboxes[child.Path]; ok {
				check.SetChecked(selected)
			}
			
			// Only recurse one level deeper to avoid freezing
			if child.IsDir && len(child.Children) > 0 {
				for _, grandchild := range child.Children {
					grandchild.Selected = selected
					if check, ok := t.checkboxes[grandchild.Path]; ok {
						check.SetChecked(selected)
					}
				}
			}
		}
	}
}

// GetSelectedFiles returns the list of selected files
func (t *FileTreeWidget) GetSelectedFiles() []*fileutils.FileInfo {
	var selected []*fileutils.FileInfo
	t.collectSelectedFiles(t.files, &selected)
	return selected
}

// collectSelectedFiles collects selected files from a list of files
func (t *FileTreeWidget) collectSelectedFiles(files []*fileutils.FileInfo, selected *[]*fileutils.FileInfo) {
	for _, file := range files {
		if file.Selected {
			// If it's a directory and has no children loaded yet, load them now
			if file.IsDir && len(file.Children) == 0 {
				t.loadChildren(file)
			}
			
			// Add the file to the selected list
			*selected = append(*selected, file)
		}
		
		// Check children if it's a directory
		if file.IsDir && file.Children != nil {
			t.collectSelectedFiles(file.Children, selected)
		}
	}
}
