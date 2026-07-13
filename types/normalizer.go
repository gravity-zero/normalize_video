package types

type Normalizer struct {
	Title				string
	// Year found in the source filename ("" when there is none, or when the
	// year IS the title - "2012" - since it already lives in Title)
	Year				string
	NormalizeFilename 	string
	NewPath				string
	EscapedNewPath		string
}

