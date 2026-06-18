package tlcli

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// TwoFieldLine represents a line with two whitespace-separated fields.
type TwoFieldLine struct {
	Field1 string
	Field2 string
}

// ReadFileTwoFields reads a file and parses each line into two fields separated by whitespace or comma.
// Empty lines and lines with only whitespace are skipped.
// If a line has only one field, Field2 will be empty.
func ReadFileTwoFields(fn string) ([]TwoFieldLine, error) {
	lines, err := ReadFileLines(fn)
	if err != nil {
		return nil, err
	}
	ws := regexp.MustCompile(`[\s,]+`)
	var ret []TwoFieldLine
	for _, line := range lines {
		parts := ws.Split(line, 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		tfl := TwoFieldLine{Field1: parts[0]}
		if len(parts) > 1 {
			tfl.Field2 = strings.TrimSpace(parts[1])
		}
		ret = append(ret, tfl)
	}
	return ret, nil
}

func ReadFileLines(fn string) ([]string, error) {
	ret := []string{}
	file, err := os.Open(fn)
	if err != nil {
		return ret, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := scanner.Text(); t != "" {
			ret = append(ret, strings.TrimSpace(t))
		}
	}
	if err := scanner.Err(); err != nil {
		return ret, err
	}
	return ret, nil
}
