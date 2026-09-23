package metadata

// Reuse contains one provisional reuse selection loaded by the command. Its syntax and retained
// format are experimental and subject to change.
//   - URI: the selected original provider resource URL
//   - Record: the source generation record, when one was supplied
type Reuse struct {
	URI    string
	Record *Record
}
