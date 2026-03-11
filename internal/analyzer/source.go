package analyzer

import "os"

// readFileBytes returns the raw contents of a file.
func readFileBytes(filename string) []byte {
	data, readErr := os.ReadFile(filename)
	if readErr != nil {
		return nil
	}

	return data
}
