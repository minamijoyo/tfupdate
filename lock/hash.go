package lock

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/sumdb/dirhash"
)

// zipDataToH1Hash is a helper function that calculates the h1 hash value from
// bytes sequence of the provider's zip archive.
func zipDataToH1Hash(zipData []byte) (string, error) {
	tmpZipfile, err := writeTempFile(zipData)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpZipfile.Name())

	// The h1 hash value in .terraform.lock.hcl uses the same hash function as go.sum.
	hash, err := dirhash.HashZip(tmpZipfile.Name(), dirhash.Hash1)
	if err != nil {
		return "", fmt.Errorf("failed to calculate h1 hash: %s", err)
	}

	return hash, nil
}

// writeTempFile writes content to a temporary file and return its file.
func writeTempFile(content []byte) (*os.File, error) {
	tmpfile, err := os.CreateTemp("", "tmp")
	if err != nil {
		return tmpfile, fmt.Errorf("failed to create temporary file: %s", err)
	}

	if _, err := tmpfile.Write(content); err != nil {
		return tmpfile, fmt.Errorf("failed to write temporary file: %s", err)
	}

	if err := tmpfile.Close(); err != nil {
		return tmpfile, fmt.Errorf("failed to close temporary file: %s", err)
	}

	return tmpfile, nil
}

// reProviderZipfile is a regular expression for parsing zip file names from shaSumsData.
// terraform-provider-null_3.2.1_darwin_arm64.zip
var reProviderZipfile = regexp.MustCompile(`terraform-provider-(?P<Type>[^_]+)+_(?P<Version>[^_]+)+_(?P<OS>[^_]+)_(?P<Arch>[^_]+)\.zip`)

// shaSumsDataToZhHash is a helper function for parsing zh hash values from
// bytes sequence of the shaSumsData document.
func shaSumsDataToZhHash(shaSumsData []byte) (map[string]string, error) {
	document := string(shaSumsData)
	zh := make(map[string]string)
	// Read an entry per line.
	for _, line := range strings.Split(document, "\n") {
		// We expect that blank lines are not normally included, but to make the
		// test data easier to read, ignore blank lines.
		if len(line) == 0 {
			continue
		}

		// Split rows into columns with spaces, but note that there are two spaces between the columns.
		// e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2  terraform-provider-null_3.2.1_darwin_arm64.zip

		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("failed to parse hash in shaSumsData: %s", document)
		}
		hash := fields[0]
		filename := fields[1]

		// Default value is the filename as it contains manifest.json and cannot be parsed.
		platform := filename

		// parse a platform name
		matches := reProviderZipfile.FindStringSubmatch(filename)
		if len(matches) != 0 {
			os := matches[reProviderZipfile.SubexpIndex("OS")]
			arch := matches[reProviderZipfile.SubexpIndex("Arch")]
			platform = os + "_" + arch
		}
		// As the implementation of the h1 hash includes a prefix for the "h1:"
		// scheme, zh also includes the "zh:" prefix for consistency.
		zh[platform] = "zh:" + hash
	}

	return zh, nil
}
