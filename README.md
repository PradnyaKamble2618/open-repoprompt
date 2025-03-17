# OpenPrompt

![OpenPrompt Interface](home.png)

OpenPrompt is a fast, lightweight Go tool that helps you feed your code and documentation to LLMs like Claude, GPT-4, and Grok without hitting token limits or waiting forever.

## Why OpenPrompt?

- **Speed**: Process gigabytes of code instantly with Go's powerful concurrency
- **Simplicity**: Select files with a simple UI, generate XML prompts with one click
- **Efficiency**: Stop wasting time on slow tools that choke on large codebases

## Features

- **Easy File Selection**: Browse and select files with a tree view and checkboxes
- **Smart Filtering**: Include/exclude files by extension, name pattern, or custom patterns
- **XML Prompt Generation**: Create perfectly formatted prompts for any LLM
- **Token Estimation**: Know exactly how many tokens you're using
- **Clipboard Integration**: Copy prompts directly to your clipboard
- **Settings Persistence**: Your preferences are saved automatically

## Use Cases

- **Code Reviews**: Feed your entire codebase for comprehensive reviews
- **Documentation Generation**: Create docs for your project based on source code
- **Refactoring Help**: Get suggestions for improving complex code
- **Bug Hunting**: Let LLMs analyze your code to find potential issues
- **Learning New Codebases**: Quickly understand unfamiliar projects
- **Architecture Analysis**: Get insights on your project structure

## Installation

### Download Pre-built Binary (Recommended)

1. Go to the [Releases page](https://github.com/wildberry-source/open-repoprompt/releases)
2. Download the appropriate version for your operating system:
   - Windows: `openprompt-windows-amd64.exe`
   - macOS: `openprompt-macos-amd64`
   - Linux: `openprompt-linux-amd64`
3. For macOS and Linux users, make the file executable:
   ```bash
   chmod +x openprompt-*-amd64
   ```
4. Double-click the executable or run it from the terminal to start OpenPrompt

### Building from Source

```bash
# Clone the repository
git clone https://github.com/wildberry-source/open-repoprompt.git

# Navigate to the project directory
cd open-repoprompt

# Build the application
go build -o openprompt ./cmd
```

## How to Use

1. **Select Directory**: Choose a directory containing your code
2. **Set Filters**: Specify which files to include/exclude
3. **Select Files**: Check the boxes next to files you want
4. **Add Instructions**: Tell the LLM what you need
5. **Generate & Copy**: Create your XML prompt and copy to clipboard
6. **Paste into LLM**: Use with Claude, GPT-4, Grok, or any LLM that accepts XML

## XML Prompt Format

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
