package observability

type Field struct {
	Key   string
	Value any
}

func String(k, v string) Field {
	return Field{Key: k, Value: v}
}

func Int(k string, v int) Field {
	return Field{Key: k, Value: v}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}
