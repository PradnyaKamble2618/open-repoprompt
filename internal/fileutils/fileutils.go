The fixed file content is below. The key change is in the `shouldIgnore` method — the pattern matching logic was restructured into four clear branches:

1. **`**` patterns**: Handle double-star globs like `logs/**` and `**/foo` correctly.
2. **Patterns with `/`**: Anchored to root — matched as prefix against the full relative path.
3. **Patterns with glob wildcards (`*`, `?`, `[`) but no `/`**: Match against the basename and every path component.
4. **Simple patterns (no wildcards, no slashes)**: e.g., `node_modules`, `.env` — now match against the **basename** and **every component** of the path, not just an exact full-path comparison. This is the critical fix: `node_modules` now correctly matches `src/node_modules/foo.js`, and `.env` matches `subdir/.env`.

```go
package fileutils

import (
	"bufio"
	"fmt"
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

// gitignorePatterns holds the patterns from .gitignore files
type gitignorePatterns struct {
	patterns []string
}

// newGitignorePatterns creates a new gitignorePatterns instance
func newGitignorePatterns(rootDir string) (*gitignorePatterns, error) {
	gitignore := &gitignorePatterns{
		patterns: []string{},
	}

	// Load the .gitignore file from the root directory
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		if err := gitignore.loadGitignoreFile(gitignorePath); err != nil {
			return nil, err
		}
	}

	return gitignore, nil
}

// loadGitignoreFile loads patterns from a .gitignore file
func (g *gitignorePatterns) loadGitignoreFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Add the pattern
		g.patterns = append(g.patterns, line)
	}

	return scanner.Err()
}

// shouldIgnore checks if a file should be ignored based on gitignore patterns
func (g *gitignorePatterns) shouldIgnore(path string, isDir bool) bool {
	// Convert path to use forward slashes for consistency with gitignore patterns
	path = filepath.ToSlash(path)

	for _, pattern := range g.patterns {
		// Handle negation (patterns starting with !)
		negate := false
		if strings.HasPrefix(pattern, "!") {
			negate = true
			pattern = pattern[1:]
		}

		// Handle directory-specific patterns (ending with /)
		dirOnly := false
		if strings.HasSuffix(pattern, "/") {
			dirOnly = true
			pattern = pattern[:len(pattern)-1]
		}

		// Skip directory-only patterns if this is a file
		if dirOnly && !isDir {
			continue
		}

		matched := false

		if strings.Contains(pattern, "**") {
			// Handle ** pattern (match any number of directories)
			// e.g., "logs/**" matches anything inside logs/
			// e.g., "**/foo" matches foo anywhere
			prefix := strings.TrimSuffix(pattern, "/**")
			if prefix != pattern {
				// Pattern like "logs/**" - match anything under that directory
				if path == prefix || strings.HasPrefix(path, prefix+"/") {
					matched = true
				}
			}
			suffix := strings.TrimPrefix(pattern, "**/")
			if !matched && suffix != pattern {
				// Pattern like "**/foo" - match foo anywhere in the tree
				if path == suffix || strings.HasSuffix(path, "/"+suffix) {
					matched = true
				}
				// Also try matching each path component
				if !matched {
					if m, _ := filepath.Match(suffix, filepath.Base(path)); m {
						matched = true
					}
				}
			}
		} else if strings.Contains(pattern, "/") {
			// Pattern contains a slash: it's anchored to the root.
			// Match as a prefix or exact match against the relative path.
			if path == pattern || strings.HasPrefix(path, pattern+"/") {
				matched = true
			}
		} else if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") || strings.Contains(pattern, "[") {
			// Pattern with glob wildcards but no slash: match against basename
			// of every path component. e.g., "*.log" matches "dir/foo.log".
			if m, _ := filepath.Match(pattern, filepath.Base(path)); m {
				matched = true
			}
			// Also check if any directory component matches (for patterns like "node_*")
			if !matched {
				parts := strings.Split(path, "/")
				for _, part := range parts {
					if m, _ := filepath.Match(pattern, part); m {
						matched = true
						break
					}
				}
			}
		} else {
			// Simple pattern without wildcards or slashes (e.g., "node_modules", ".env").
			// Per gitignore spec, this should match against the basename of the path
			// AND against any path component, so "node_modules" ignores
			// "node_modules/", "src/node_modules/foo.js", etc.
			baseName := filepath.Base(path)
			if baseName == pattern {
				matched = true
			}
			// Also check if any directory component in the path matches
			if !matched {
				parts := strings.Split(path, "/")
				for _, part := range parts {
					if part == pattern {
						matched = true
						break
					}
				}
			}
		}

		// If the pattern matches, respect negation
		if matched {
			return !negate
		}
	}

	return false
}

// FileFilters represents filters for file selection
type FileFilters struct {
	Extensions     []string
	NamePattern    string
	IgnorePatterns []string
	SubPath        string // Path relative to the root directory
	RespectGitignore bool // Whether to respect .gitignore files
}

// ListFiles returns a list of files in the given directory
func ListFiles(dir string, filters FileFilters) ([]*FileInfo, error) {
	var result []*FileInfo

	// If SubPath is specified, adjust the directory
	if filters.SubPath != "" {
		dir = filepath.Join(dir, filters.SubPath)
	}

	// Load gitignore patterns if needed
	var gitignore *gitignorePatterns
	var err error
	if filters.RespectGitignore {
		gitignore, err = newGitignorePatterns(dir)
		if err != nil {
			return nil, err
		}
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		// Get relative path for gitignore checking
		relPath, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			relPath = path
		}

		// Skip if matches gitignore patterns
		if filters.RespectGitignore && gitignore != nil && path != dir {
			if gitignore.shouldIgnore(relPath, info.IsDir()) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
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

// FormatTokenCount formats a token count into a human-readable string (e.g., 1.2K, 3.5M)
func FormatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	} else if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
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
```