package types

type Serie struct {
	*Video
	SE 			string
	Saison		string
	Episode		string
	SaisonPath	string
	Normalizer
	MkvMetadata FileInfos
}

func (s *Serie) GetVideo() *Video {
	return s.Video
}

func (s *Serie) GetNormalizer() *Normalizer {
	return &s.Normalizer
}

func (s *Serie) GetSE() string {
	return s.SE
}

func (s *Serie) SetTitle(title string) {
	s.Normalizer.Title = title
}

func (s *Serie) SetNormalizeFilename(filename string) {
	s.Normalizer.NormalizeFilename = filename
}

func (s *Serie) SetNewPath(path string) {
	s.Normalizer.NewPath = path
}

func (s *Serie) SetEscapedNewPath(escapedPath string) {
	s.Normalizer.EscapedNewPath = escapedPath
}