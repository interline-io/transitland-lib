package cli

import "strings"

// ArrayFlags allow repeatable command line options.
// https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang/28323276#28323276
type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
