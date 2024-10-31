package tlcli

import (
	"bufio"
	"os"
	"strings"
)

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
