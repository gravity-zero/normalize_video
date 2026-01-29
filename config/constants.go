package config

const (
    ORIGIN_PATH = "/mnt/e/DDL/"
    DEST_PATH = "/mnt/e/Cartoon/"
	RECURSIVE_SCAN = true

	REGEXSERIES = `\b[sS]\s*(\d{1,2})\s*[-._ ]*\s*[eE]\s*(\d{1,3})\b`
    REGEXSERIESEXTEND = `\b(\d{1,2})\s*[xX]\s*(\d{1,3})\b`

    PREFERRED_AUDIO_LANG    = "fr"
    PREFERRED_SUBTITLE_LANG = "fr"

    FALLBACK_AUDIO_LANG     = ""
    FALLBACK_SUBTITLE_LANG  = ""

    SUBTITLE_FORCED_ONLY = true

    MAX_WORKERS = 10
)

var Extensions = []string{
	"avi", "mkv", "mp4", "mpeg", "mpg",
	"mov", "wmv", "flv", "webm",
	"m4v", "3gp", "ogv",
	"ts", "mts", "m2ts",
}

var Qualities = []string{
	"480p", "720p", "1080p", "2160p", "4k", "8k", "uhd",
	"cam", "hdcam", "ts", "telesync", "screener",
	"dvdrip", "bdrip", "brrip", "hdtv",
	"web", "webdl", "web-dl", "webrip",
	"bluray", "remux",
}