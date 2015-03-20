package main

// Split a shell-escaped string (basic implementation)
func ShellSplit(escaped string) []string {
	var result []string
	field := ""
	escaping := false

	for _, current := range escaped {
		if current == '\\' && !escaping {
			escaping = true
		} else if escaping {
			escaping = false
			switch current {
			case 'r':
				field += "\r"
			case 'n':
				field += "\n"
			case 't':
				field += "\t"
			default:
				field += string(current)
			}
		} else if current == ' ' {
			if len(field) > 0 {
				result = append(result, field)
				field = ""
			}
		} else {
			field += string(current)
		}
	}
	if len(field) > 0 {
		result = append(result, field)
	}
	return result
}
