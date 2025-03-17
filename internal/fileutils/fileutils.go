package fileutils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileInfo represents a file or directory in the file system
type FileInfo struct {
	Path       string
	Name       string
	IsDir      bool
	Size       int64
	Extension  string
	Selected   bool
	Children   []*FileInfo
	TokenCount int // Estimated token count
}

// ListFiles returns a list of files in the given directory
func ListFiles(dir string, filters FileFilters) ([]*FileInfo, error) {
	var result []*FileInfo
	
	// If SubPath is specified, adjust the directory
	if filters.SubPath != "" {
		dir = filepath.Join(dir, filters.SubPath)
	}
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}
		
		// Skip if matches ignore patterns
		for _, pattern := range filters.IgnorePatterns {
			matched, err := filepath.Match(pattern, info.Name())
			if err == nil && matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		
		// For files, check extension and name pattern filters
		if !info.IsDir() {
			// Check extension filter
			if len(filters.Extensions) > 0 {
				ext := filepath.Ext(path)
				if ext != "" {
					ext = ext[1:] // Remove leading dot
				}
				
				found := false
				for _, e := range filters.Extensions {
					if e == ext {
						found = true
						break
					}
				}
				
				if !found {
					return nil
				}
			}
			
			// Check name pattern filter
			if filters.NamePattern != "" {
				matched, err := filepath.Match(filters.NamePattern, info.Name())
				if err != nil || !matched {
					return nil
				}
			}
		}
		
		fileInfo := &FileInfo{
			Path:      path,
			Name:      info.Name(),
			IsDir:     info.IsDir(),
			Size:      info.Size(),
			Extension: filepath.Ext(path),
			Selected:  false,
			TokenCount: int(info.Size() / 4), // Rough estimate: 1 token per 4 characters
		}
		
		result = append(result, fileInfo)
		return nil
	})
	
	return result, err
}

// FileFilters represents filters for file selection
type FileFilters struct {
	Extensions     []string
	NamePattern    string
	IgnorePatterns []string
	SubPath        string // Path relative to the root directory
}

// ParseExtensions parses a comma-separated list of extensions
func ParseExtensions(input string) []string {
	if input == "" {
		return nil
	}
	
	extensions := strings.Split(input, ",")
	for i, ext := range extensions {
		extensions[i] = strings.TrimSpace(ext)
	}
	
	return extensions
}

// ParseIgnorePatterns parses a comma-separated list of ignore patterns
func ParseIgnorePatterns(input string) []string {
	if input == "" {
		return nil
	}
	
	patterns := strings.Split(input, ",")
	for i, pattern := range patterns {
		patterns[i] = strings.TrimSpace(pattern)
	}
	
	return patterns
}

// GetSelectedFiles returns a list of selected files
func GetSelectedFiles(files []*FileInfo) []*FileInfo {
	var selected []*FileInfo
	
	for _, file := range files {
		if file.Selected && !file.IsDir {
			selected = append(selected, file)
		}
		
		if len(file.Children) > 0 {
			selected = append(selected, GetSelectedFiles(file.Children)...)
		}
	}
	
	return selected
}

// BuildFileTree builds a hierarchical file tree from a flat list of files
func BuildFileTree(files []*FileInfo) []*FileInfo {
	// Map to store directories
	dirMap := make(map[string]*FileInfo)
	
	// Root of the tree
	var root []*FileInfo
	
	// First pass: create all directories and initialize their Children slices
	for _, file := range files {
		// Initialize Children slice for all files
		if file.Children == nil {
			file.Children = []*FileInfo{}
		}
		
		if file.IsDir {
			dirMap[file.Path] = file
		}
	}
	
	// Second pass: add files to their parent directories
	for _, file := range files {
		if file.IsDir {
			// Skip directories for now, we'll handle them in the third pass
			continue
		}
		
		// Get parent directory
		parentPath := filepath.Dir(file.Path)
		
		// If parent is in the map, add file to its children
		if parent, ok := dirMap[parentPath]; ok {
			parent.Children = append(parent.Children, file)
		} else {
			// If parent is not in the map, add file to root
			root = append(root, file)
		}
	}
	
	// Third pass: build directory hierarchy
	for _, dir := range dirMap {
		// Skip the current directory if it's already in the root
		alreadyInRoot := false
		for _, rootDir := range root {
			if rootDir.Path == dir.Path {
				alreadyInRoot = true
				break
			}
		}
		if alreadyInRoot {
			continue
		}
		
		// Get parent directory
		parentPath := filepath.Dir(dir.Path)
		
		// If this is the root directory or parent path is the same as current path
		if parentPath == dir.Path || parentPath == "." {
			root = append(root, dir)
			continue
		}
		
		// If parent is in the map, add directory to its children
		if parent, ok := dirMap[parentPath]; ok && parent.Path != dir.Path {
			// Avoid circular references
			parent.Children = append(parent.Children, dir)
		} else {
			// If parent is not in the map, add directory to root
			root = append(root, dir)
		}
	}
	
	// If root is empty but we have files, something went wrong
	// Add all directories to root as a fallback
	if len(root) == 0 && len(files) > 0 {
		for _, file := range files {
			if file.IsDir {
				root = append(root, file)
			}
		}
		
		// If still empty, add all files
		if len(root) == 0 {
			root = files
		}
	}
	
	// Sort root items so directories come first
	sortFileTreeDirectoriesFirst(root)
	
	// Sort children of all directories
	for _, file := range files {
		if file.IsDir && len(file.Children) > 0 {
			sortFileTreeDirectoriesFirst(file.Children)
		}
	}
	
	return root
}

// CalculateDirectoryTokenCount calculates the total token count for a directory based on its children
func CalculateDirectoryTokenCount(dir *FileInfo) int {
	if !dir.IsDir {
		return dir.TokenCount
	}
	
	totalTokens := 0
	for _, child := range dir.Children {
		if child.IsDir {
			totalTokens += CalculateDirectoryTokenCount(child)
		} else {
			totalTokens += child.TokenCount
		}
	}
	
	// Update the directory's token count
	dir.TokenCount = totalTokens
	
	return totalTokens
}

// sortFileTreeDirectoriesFirst sorts a slice of FileInfo so directories come first
func sortFileTreeDirectoriesFirst(files []*FileInfo) {
	// Sort the files so directories come first, then by name
	sort.Slice(files, func(i, j int) bool {
		// If one is a directory and the other is not, the directory comes first
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		// Otherwise, sort by name
		return files[i].Name < files[j].Name
	})
}
