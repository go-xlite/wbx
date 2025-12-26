package comm

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
	".wasm": "application/wasm",
	".bin":  "application/octet-stream",
	".map":  "application/json",
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
