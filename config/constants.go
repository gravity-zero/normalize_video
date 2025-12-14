package config

const (
    ORIGIN_PATH = "/mnt/e/DDL/"
    DEST_PATH = "/mnt/e/Cartoon/"
	REGEXSERIES = `\b[sS]\s*(\d{1,2})\s*[-._ ]*\s*[eE]\s*(\d{1,3})\b`
    REGEXSERIESEXTEND = `\b(\d{1,2})\s*[xX]\s*(\d{1,3})\b`
)

var Extensions = []string{"avi", "mkv", "mp4", "mpeg"}
var Languages = []string{"vostfr", "vf", "vff", "vfi", "french", "truefrench", "vo", "multi"}
var Qualities = []string{"cam", "hdcam", "bdrip", "dvdrip", "hdtv", "720p", "1080p", "2160p", "4k", "8k"}
