package errs

// ExcerptLimit is the maximum number of bytes included in a diagnostic excerpt.
const ExcerptLimit = 200

// Excerpt returns a bounded prefix of encoded data for diagnostic messages.
func Excerpt(encodedData []byte) string {
	if len(encodedData) > ExcerptLimit {
		return string(encodedData[:ExcerptLimit])
	}

	return string(encodedData)
}
