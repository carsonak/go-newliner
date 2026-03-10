package analyzer

import "os"

// readFileBytes returns the raw contents of a file.
func readFileBytes(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	return data
}
