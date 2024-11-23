package tt

type Phone struct {
	Option[string]
}

func NewPhone(v string) Phone {
	return Phone{Option: NewOption(v)}
}

func (r Phone) Check() error {
	return nil
}
