package core

// SliceContainsString takes in a slice of strings and a string to check for
// within the supplied slice of strings, then returns a bool indicating if the
// the specified string was found and the index where it was found. If the
// specified string was not found then the index returned is -1.
func SliceContainsString(slice []string, toFind string) (bool, int) {
	for idx, str := range slice {
		if str == toFind {
			return true, idx
		}
	}
	return false, -1
}
