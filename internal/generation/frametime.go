package generation

// numericSeconds returns a numeric parameter as float64.
func numericSeconds(paramValue any) (float64, bool) {
	switch value := paramValue.(type) {
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case float64:
		return value, true
	}

	return 0, false
}
