package metadata

// Reuse contains one provisional reuse selection loaded by the command.
// Its syntax and retained format are experimental and subject to change.
type Reuse struct {
	URI    string
	Record *Record
}
