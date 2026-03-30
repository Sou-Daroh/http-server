package server

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler returns a HandlerFunc that serves files from root directory.
// It protects against path traversal attacks and sets correct MIME types.
func StaticHandler(root string) HandlerFunc {
	return func(req *Request, res *ResponseWriter) {
		serveStatic(root, req, res)
	}
}

// serveStatic resolves the request path to a file on disk and serves it.
func serveStatic(root string, req *Request, res *ResponseWriter) {
	// Clean and join path — filepath.Clean prevents traversal attacks
	// by resolving ".." and "." segments.
	cleanPath := filepath.Clean(filepath.Join(root, req.Path))

	// Ensure the resolved path is still within root
	absRoot, err := filepath.Abs(root)
	if err != nil {
		res.WriteText(500, "Internal Server Error")
		return
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		res.WriteText(400, "Bad Request")
		return
	}

	if !strings.HasPrefix(absPath, absRoot) {
		// Path traversal attempt
		res.WriteText(400, "Bad Request")
		return
	}

	// Stat the path
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			res.WriteText(404, "Not Found")
			return
		}
		res.WriteText(500, "Internal Server Error")
		return
	}

	// If it's a directory, look for index.html
	if info.IsDir() {
		indexPath := filepath.Join(absPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			absPath = indexPath
			info, _ = os.Stat(absPath)
		} else {
			serveDirectoryListing(absPath, req.Path, res)
			return
		}
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		res.WriteText(500, "Internal Server Error")
		return
	}

	// Detect MIME type
	ext := strings.ToLower(filepath.Ext(absPath))
	contentType := mimeTypeForExt(ext)

	res.WriteHeader(200)
	res.SetHeader("Content-Type", contentType)
	res.SetHeader("Last-Modified", info.ModTime().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	res.Write(data)
}

// serveDirectoryListing renders a simple HTML directory listing.
func serveDirectoryListing(dirPath, urlPath string, res *ResponseWriter) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		res.WriteText(500, "Internal Server Error")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Index of %s</title>
<style>
  body { font-family: monospace; padding: 2rem; }
  a { display: block; margin: 0.25rem 0; }
</style>
</head>
<body>
<h2>Index of %s</h2>
<hr>
`, urlPath, urlPath))

	// Parent directory link
	if urlPath != "/" {
		sb.WriteString(`<a href="../">../</a>`)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>\n", name, name))
	}

	sb.WriteString("<hr></body></html>")

	res.WriteHTML(200, sb.String())
}

// mimeTypeForExt returns the MIME type for a file extension.
// Falls back to mime.TypeByExtension, then "application/octet-stream".
func mimeTypeForExt(ext string) string {
	// Common types — explicit for reliability
	known := map[string]string{
		".html": "text/html; charset=utf-8",
		".htm":  "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".txt":  "text/plain; charset=utf-8",
		".md":   "text/markdown",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".woff": "font/woff",
		".woff2": "font/woff2",
		".ttf":  "font/ttf",
		".otf":  "font/otf",
	}

	if t, ok := known[ext]; ok {
		return t
	}

	// Fall back to system MIME database
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}

	return "application/octet-stream"
}
