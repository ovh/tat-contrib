package utils

// ArrayContains return true if element is in array
func ArrayContains(array []string, element string) bool {
	for _, cur := range array {
		if cur == element {
			return true
		}
	}
	return false
}
