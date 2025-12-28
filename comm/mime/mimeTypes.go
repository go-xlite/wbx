package mime

import "strings"

// MIME type mapping
var mimeTypes = map[string]string{
	// Text
	".html": "text/html; charset=utf-8",
	".htm":  "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".txt":  "text/plain; charset=utf-8",
	".xml":  "text/xml; charset=utf-8",
	".csv":  "text/csv; charset=utf-8",

	// JavaScript
	".js":   "application/javascript; charset=utf-8",
	".mjs":  "application/javascript; charset=utf-8",
	".json": "application/json; charset=utf-8",

	// Images
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".avif": "image/avif",

	// Fonts
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".eot":   "application/vnd.ms-fontobject",

	// Video
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".ogg":  "video/ogg",
	".ogv":  "video/ogg",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".mkv":  "video/x-matroska",
	".m4v":  "video/x-m4v",

	// Audio
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".oga":  "audio/ogg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".flac": "audio/flac",

	// Documents
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

	// Archives
	".zip": "application/zip",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
	".bz2": "application/x-bzip2",
	".7z":  "application/x-7z-compressed",
	".rar": "application/vnd.rar",

	// Other
	".wasm":     "application/wasm",
	".bin":      "application/octet-stream",
	".map":      "application/json",
	".manifest": "application/manifest+json",
}

var staticExtensions = []string{
	".html", ".htm", ".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg",
	".webp", ".woff", ".woff2", ".ttf", ".otf", ".eot", ".mp4", ".webm",
	".mp3", ".wav", ".pdf", ".zip", ".tar", ".gz", ".wasm",
}

func IsStaticExtension(ext string) bool {
	ext = strings.ToLower(ext)
	for _, v := range staticExtensions {
		if v == ext {
			return true
		}
	}
	return false
}

// GetMimeType returns the MIME type for a file extension
func GetMimeType(ext string) string {
	ext = strings.ToLower(ext)
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

type MimeTypes struct {
	// Text
	Html string
	Htm  string
	Css  string
	Txt  string
	Xml  string
	Csv  string

	// JavaScript
	Js   string
	Mjs  string
	Json string

	// Images
	Jpg  string
	Jpeg string
	Png  string
	Gif  string
	Svg  string
	Webp string
	Ico  string
	Bmp  string
	Tiff string
	Tif  string
	Avif string

	// Fonts
	Woff  string
	Woff2 string
	Ttf   string
	Otf   string
	Eot   string

	// Video
	Mp4  string
	Webm string
	Ogg  string
	Ogv  string
	Avi  string
	Mov  string
	Mkv  string
	M4v  string

	// Audio
	Mp3  string
	Wav  string
	Oga  string
	M4a  string
	Aac  string
	Flac string

	// Documents
	Pdf  string
	Doc  string
	Docx string
	Xls  string
	Xlsx string
	Ppt  string
	Pptx string

	// Archives
	Zip string
	Tar string
	Gz  string
	Bz2 string
	Z7z string
	Rar string

	// Other
	Wasm     string
	Bin      string
	Map      string
	Manifest string
}

var Mime = MimeTypes{
	// Text
	Html: mimeTypes[".html"],
	Htm:  mimeTypes[".htm"],
	Css:  mimeTypes[".css"],
	Txt:  mimeTypes[".txt"],
	Xml:  mimeTypes[".xml"],
	Csv:  mimeTypes[".csv"],

	// JavaScript
	Js:   mimeTypes[".js"],
	Mjs:  mimeTypes[".mjs"],
	Json: mimeTypes[".json"],

	// Images
	Jpg:  mimeTypes[".jpg"],
	Jpeg: mimeTypes[".jpeg"],
	Png:  mimeTypes[".png"],
	Gif:  mimeTypes[".gif"],
	Svg:  mimeTypes[".svg"],
	Webp: mimeTypes[".webp"],
	Ico:  mimeTypes[".ico"],
	Bmp:  mimeTypes[".bmp"],
	Tiff: mimeTypes[".tiff"],
	Tif:  mimeTypes[".tif"],
	Avif: mimeTypes[".avif"],

	// Fonts
	Woff:  mimeTypes[".woff"],
	Woff2: mimeTypes[".woff2"],
	Ttf:   mimeTypes[".ttf"],
	Otf:   mimeTypes[".otf"],
	Eot:   mimeTypes[".eot"],

	// Video
	Mp4:  mimeTypes[".mp4"],
	Webm: mimeTypes[".webm"],
	Ogg:  mimeTypes[".ogg"],
	Ogv:  mimeTypes[".ogv"],
	Avi:  mimeTypes[".avi"],
	Mov:  mimeTypes[".mov"],
	Mkv:  mimeTypes[".mkv"],
	M4v:  mimeTypes[".m4v"],

	// Audio
	Mp3:  mimeTypes[".mp3"],
	Wav:  mimeTypes[".wav"],
	Oga:  mimeTypes[".oga"],
	M4a:  mimeTypes[".m4a"],
	Aac:  mimeTypes[".aac"],
	Flac: mimeTypes[".flac"],

	// Documents
	Pdf:  mimeTypes[".pdf"],
	Doc:  mimeTypes[".doc"],
	Docx: mimeTypes[".docx"],
	Xls:  mimeTypes[".xls"],
	Xlsx: mimeTypes[".xlsx"],
	Ppt:  mimeTypes[".ppt"],
	Pptx: mimeTypes[".pptx"],

	// Archives
	Zip: mimeTypes[".zip"],
	Tar: mimeTypes[".tar"],
	Gz:  mimeTypes[".gz"],
	Bz2: mimeTypes[".bz2"],
	Z7z: mimeTypes[".7z"],
	Rar: mimeTypes[".rar"],

	// Other
	Wasm:     mimeTypes[".wasm"],
	Bin:      mimeTypes[".bin"],
	Map:      mimeTypes[".map"],
	Manifest: mimeTypes[".manifest"],
}
