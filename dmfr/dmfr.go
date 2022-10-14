package dmfr

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GetReaderURL helps load a file from an S3 or Directory location
func GetReaderURL(s3 string, directory string, url string, sha1 string) string {
	if s3 != "" && sha1 != "" {
		url = fmt.Sprintf("%s/%s.zip", s3, sha1)
	} else if directory != "" {
		url = filepath.Join(directory, url)
	}
	urlsplit := strings.SplitN(url, "#", 2)
	if len(urlsplit) > 1 && !strings.HasSuffix(url, ".zip") {
		// Add fragment back only if fragment does not end in ".zip"
		url = url + "#" + urlsplit[1]
	}
	return url
}
