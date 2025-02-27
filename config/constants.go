package config

const (
    ORIGIN_PATH = ""
    DEST_PATH = ""
	REGEXSERIES = `(S|s)(\d\d?)(-){0,1}(E|e)(\d\d?)+`
)

var Extensions = []string{"avi", "mkv", "mp4", "mpeg"}
var Languages = []string{"vostfr", "vf", "vff", "vfi", "french", "truefrench", "vo", "multi"}
var Qualities = []string{"cam", "hdcam", "bdrip", "dvdrip", "hdtv", "720p", "1080p", "2160p", "4k", "8k"}
