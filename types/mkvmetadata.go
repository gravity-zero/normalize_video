package types

type FileInfos struct {
	EscapedPath   string
	MkvTitle      string
	MkvAudioTrack string
	MkvSubTrack   string
}

type Metadata struct {
	Tracks []Track `json:"tracks"`
}

type Track struct {
	Type       string          `json:"type"`
	Properties TrackProperties `json:"properties"`
}

type TrackProperties struct {
	Number       int    `json:"number"`
	LanguageIetf string `json:"language_ietf"`
	Language     string `json:"language"`
	TrackName    string `json:"track_name"`
}
