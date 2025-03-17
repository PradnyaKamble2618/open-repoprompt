package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences represents user preferences
type Preferences struct {
	LastDirectory string            `json:"lastDirectory"`
	Filters       map[string]string `json:"filters"`
}

// DefaultPreferences returns default preferences
func DefaultPreferences() *Preferences {
	return &Preferences{
		LastDirectory: "",
		Filters:       make(map[string]string),
	}
}

// GetPreferencesDir returns the directory where preferences are stored
func GetPreferencesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	prefsDir := filepath.Join(homeDir, ".openprompt")
	
	// Create directory if it doesn't exist
	if _, err := os.Stat(prefsDir); os.IsNotExist(err) {
		err = os.MkdirAll(prefsDir, 0755)
		if err != nil {
			return "", err
		}
	}
	
	return prefsDir, nil
}

// GetPreferencesFile returns the path to the preferences file
func GetPreferencesFile() (string, error) {
	prefsDir, err := GetPreferencesDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(prefsDir, "preferences.json"), nil
}

// Load loads preferences from disk
func Load() (*Preferences, error) {
	prefsFile, err := GetPreferencesFile()
	if err != nil {
		return DefaultPreferences(), err
	}
	
	// Check if file exists
	if _, err := os.Stat(prefsFile); os.IsNotExist(err) {
		// Return default preferences if file doesn't exist
		return DefaultPreferences(), nil
	}
	
	// Read file
	data, err := os.ReadFile(prefsFile)
	if err != nil {
		return DefaultPreferences(), err
	}
	
	// Parse JSON
	prefs := DefaultPreferences()
	err = json.Unmarshal(data, prefs)
	if err != nil {
		return DefaultPreferences(), err
	}
	
	return prefs, nil
}

// Save saves preferences to disk
func (p *Preferences) Save() error {
	prefsFile, err := GetPreferencesFile()
	if err != nil {
		return err
	}
	
	// Convert to JSON
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file
	return os.WriteFile(prefsFile, data, 0644)
}

// SetLastDirectory sets the last directory
func (p *Preferences) SetLastDirectory(dir string) {
	p.LastDirectory = dir
}

// GetLastDirectory gets the last directory
func (p *Preferences) GetLastDirectory() string {
	return p.LastDirectory
}

// SetFilter sets a filter
func (p *Preferences) SetFilter(name, value string) {
	p.Filters[name] = value
}

// GetFilter gets a filter
func (p *Preferences) GetFilter(name string) string {
	return p.Filters[name]
}
