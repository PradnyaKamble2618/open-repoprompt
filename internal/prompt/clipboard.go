package prompt

import (
	"github.com/atotto/clipboard"
)

// CopyToClipboard copies text to the clipboard
func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}
