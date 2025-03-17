package prompt

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/openprompt/internal/fileutils"
	"github.com/pkoukk/tiktoken-go"
	gitignore "github.com/sabhiram/go-gitignore"
)

// Prompt represents an XML prompt for an LLM
type Prompt struct {
	XMLName      xml.Name `xml:"prompt"`
	Files        []File   `xml:"files>file"`
	Instructions string   `xml:"instructions"`
}

// File represents a file in the XML prompt
type File struct {
	Path    string `xml:"path,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:"filecontents"`
}

// fileReadResult represents the result of reading a file
type fileReadResult struct {
	file File
	err  error
}

// bufferPool is a pool of byte buffers for reading files
var bufferPool = sync.Pool{
	New: func() interface{} {
		// Create a reasonably sized buffer (64KB)
		buffer := make([]byte, 64*1024)
		return &buffer
	},
}

// GenerateXML generates an XML prompt from a list of files
func GenerateXML(files []*fileutils.FileInfo, instructions string, baseDir string) (string, error) {
	// Start with maximum parallelism
	runtime.GOMAXPROCS(runtime.NumCPU())

	prompt := Prompt{
		Instructions: instructions,
	}

	// Count how many files we need to process (non-directories)
	fileCount := 0
	for _, file := range files {
		if !file.IsDir {
			fileCount++
		}
	}

	if fileCount == 0 {
		// No files to process
		xmlData, err := xml.MarshalIndent(prompt, "", "  ")
		if err != nil {
			return "", err
		}
		return xml.Header + string(xmlData), nil
	}

	// Create a channel to collect results
	resultChan := make(chan fileReadResult, fileCount)

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Determine optimal number of workers based on CPU count
	numWorkers := runtime.NumCPU() * 2 // Use more workers than CPUs to maximize I/O parallelism

	// Create a channel to distribute work
	workChan := make(chan *fileutils.FileInfo, fileCount)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileInfo := range workChan {
				// Skip directories
				if fileInfo.IsDir {
					continue
				}

				// Get relative path
				relPath, err := filepath.Rel(baseDir, fileInfo.Path)
				if err != nil {
					relPath = fileInfo.Path
				}

				// Skip .DS_Store files
				if strings.HasSuffix(relPath, ".DS_Store") {
					continue
				}

				// Check if file should be ignored based on .gitignore
				gitIgnorePath := filepath.Join(baseDir, ".gitignore")
				if _, err := os.Stat(gitIgnorePath); err == nil {
					ignore, err := gitignore.CompileIgnoreFile(gitIgnorePath)
					if err == nil && ignore.MatchesPath(relPath) {
						continue
					}
				}

				// Get file type
				fileType := filepath.Ext(fileInfo.Path)
				if fileType != "" {
					fileType = fileType[1:] // Remove leading dot
				}

				// Get a buffer from the pool
				bufPtr := bufferPool.Get().(*[]byte)
				buffer := *bufPtr

				// Read file content using buffered I/O for better performance
				content, err := readFileWithBuffer(fileInfo.Path, buffer)
				if err != nil {
					// Return the buffer to the pool
					bufferPool.Put(bufPtr)

					resultChan <- fileReadResult{
						err: fmt.Errorf("error reading file %s: %v", fileInfo.Path, err),
					}
					continue
				}

				// Send result
				resultChan <- fileReadResult{
					file: File{
						Path:    relPath,
						Type:    fileType,
						Content: content,
					},
					err: nil,
				}

				// Return the buffer to the pool
				bufferPool.Put(bufPtr)
			}
		}()
	}

	// Send work to workers
	for _, file := range files {
		if !file.IsDir {
			workChan <- file
		}
	}
	close(workChan)

	// Wait for all workers in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var lastError error
	var processedCount int32 = 0

	// Pre-allocate the slice with the expected capacity
	prompt.Files = make([]File, 0, fileCount)

	// Process results as they come in
	for result := range resultChan {
		if result.err != nil {
			lastError = result.err
			fmt.Printf("Error: %v\n", result.err)
			continue
		}

		prompt.Files = append(prompt.Files, result.file)
		atomic.AddInt32(&processedCount, 1)

		// Debug output (less frequent to reduce overhead)
		if processedCount%500 == 0 {
			fmt.Printf("Processed %d/%d files\n", processedCount, fileCount)
		}
	}

	fmt.Printf("Total files processed: %d/%d\n", processedCount, fileCount)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return "", err
	}

	// Add XML header
	xmlString := xml.Header + string(xmlData)

	return xmlString, lastError
}

// readFileWithBuffer reads a file using a provided buffer for better performance
func readFileWithBuffer(filePath string, buffer []byte) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	// If the file is larger than our buffer, create a new buffer of appropriate size
	size := info.Size()
	var content []byte

	if size <= int64(len(buffer)) {
		// File fits in our buffer
		content = buffer[:size]
		_, err = io.ReadFull(file, content)
		if err != nil {
			return "", err
		}
	} else {
		// File is larger than our buffer, use a new buffer
		content = make([]byte, size)
		reader := bufio.NewReader(file)
		_, err = io.ReadFull(reader, content)
		if err != nil {
			return "", err
		}
	}

	return string(content), nil
}

// EstimateTokens estimates the number of tokens in a string
func EstimateTokens(text string) (int, error) {
	// Use tiktoken-go for accurate tokenization
	tk, err := tiktoken.GetEncoding("cl100k_base") // For GPT-3/4
	if err != nil {
		// Fallback to rough estimate if tiktoken fails
		return len(text) / 4, nil
	}

	tokens := tk.Encode(text, nil, nil)
	return len(tokens), nil
}
