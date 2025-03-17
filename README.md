# OpenPrompt

OpenPrompt is a native Go application for generating optimized XML prompts for large language models (LLMs). It allows you to select files from your filesystem, filter them based on various criteria, and generate structured XML prompts that can be copied to the clipboard for use with LLMs.

## Features

- **Advanced File Selection**: Easily select files with a tree view and checkboxes
- **Filtering Capabilities**: Filter files by extension, name pattern, and ignore patterns
- **XML Prompt Generation**: Generate optimized XML prompts for large language models
- **Token Estimation**: Estimate token usage for your prompt
- **Clipboard Integration**: Copy the generated XML prompt to your clipboard
- **Persistent Settings**: Automatically saves your last directory and filter settings
- **Manual Path Entry**: Enter directory paths directly or use the file browser
- **Fast and Native**: Built with Go and Fyne for a responsive, cross-platform experience

## Installation

### Prerequisites

- Go 1.16 or later

### Building from Source

1. Clone the repository
2. Navigate to the project directory
3. Run `go build -o openprompt ./cmd`

## Usage

1. **Select Directory**: Click the "Select Directory" button to choose a directory containing your code files. You can either browse for a directory or enter the path manually.

2. **Apply Filters**:
   - Enter file extensions to include (e.g., `go,txt,md`)
   - Specify name patterns using glob syntax (e.g., `main*` for files starting with "main")
   - Define ignore patterns for files/directories to exclude (e.g., `.git,node_modules`)
   - Click "Apply Filters" to update the file tree

3. **Select Files**: Check the boxes next to the files you want to include in your prompt

4. **Add Instructions**: Enter instructions for the LLM in the provided text area

5. **Generate Prompt**: Click "Generate XML" to create the prompt based on your selections

6. **Copy to Clipboard**: Click "Copy to Clipboard" to copy the generated XML prompt

7. **Preview XML**: Click "Preview XML" to view the generated XML prompt

Your settings, including the last directory and filter preferences, will be automatically saved and restored the next time you open the application.

## XML Prompt Format

The XML prompt format is structured as follows:

```xml
<prompt>
  <files>
    <file path="project/main.go" type="go">package main

func main() {
    println("Hello")
}
</file>
    <file path="project/utils/helper.go" type="go">...</file>
  </files>
  <instructions>Analyze the code and suggest improvements.</instructions>
</prompt>
```

## License

MIT
