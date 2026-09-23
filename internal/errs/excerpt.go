package errs

// excerptLimit is the maximum number of bytes included in a diagnostic excerpt.
const excerptLimit = 200

// Excerpt returns a bounded prefix of encoded data for diagnostic messages.
func Excerpt(encodedData []byte) string {
	if len(encodedData) > excerptLimit {
		return string(encodedData[:excerptLimit])
	}

	return string(encodedData)
}
