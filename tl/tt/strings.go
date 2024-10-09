package tt

// Strings helps read and write []String as JSON
type Strings struct {
	Option[[]string]
}

func NewStrings(v []string) Strings {
	s := Strings{}
	s.Valid = true
	s.Val = append(s.Val, v...)
	return s
}
