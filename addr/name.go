package addr

func ValidName(name string) bool {
	for _, r := range name {
		switch {
		case r >= '0' && r <= '9':
			continue
		case r >= 'a' && r <= 'z':
			continue
		case r >= 'A' && r <= 'Z':
			continue
		default:
			return false
		}
	}
	return true
}
