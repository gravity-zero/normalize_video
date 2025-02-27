package types

type Movie struct {
	*Video
	Normalizer
	MkvMetadata FileInfos
}

func (m *Movie) GetVideo() *Video {
	return m.Video
}

func (m *Movie) GetNormalizer() *Normalizer {
	return &m.Normalizer
}

func (m *Movie) GetSE() string {
	return ""
}

func (m *Movie) SetTitle(title string) {
	m.Normalizer.Title = title
}

func (m *Movie) SetNormalizeFilename(filename string) {
	m.Normalizer.NormalizeFilename = filename
}

func (m *Movie) SetNewPath(path string) {
	m.Normalizer.NewPath = path
}

func (m *Movie) SetEscapedNewPath(escapedPath string) {
	m.Normalizer.EscapedNewPath = escapedPath
}