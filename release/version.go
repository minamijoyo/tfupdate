package release

func tagNameToVersion(tagName string) string {
	// if a tagName starts with `v`, remove it.
	if tagName[0] == 'v' {
		return tagName[1:]
	}

	return tagName
}
