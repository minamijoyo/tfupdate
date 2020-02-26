package release

func tagNameToVersion(tagName string) string {
	// if a tagName starts with `v`, remove it.
	if tagName[0] == 'v' {
		return tagName[1:]
	}

	return tagName
}

func reverseStringSlice(s []string) []string {
	r := []string{}
	// apparently inefficient but simple way
	for i := len(s) - 1; i >= 0; i-- {
		r = append(r, s[i])
	}
	return r
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
