package types

type Normalizable interface {
	GetVideo() *Video
	GetNormalizer() *Normalizer
	GetSE() string //Only available for Series
	SetTitle(title string)
	SetNormalizeFilename(filename string)
	SetNewPath(path string)
	SetEscapedNewPath(escapedPath string)
}
