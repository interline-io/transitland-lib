package tt

// Key is a nullable foreign key constraint, similar to sql.NullString
type Key struct {
	Option[string]
}

func NewKey(v string) Key {
	return Key{Option[string]{Valid: true, Val: v}}
}
