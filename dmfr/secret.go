package dmfr

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// Secret holds the credentials for fetching a feed.
type Secret struct {
	Key                string `json:"key"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	AWSProfile         string `json:"aws_profile"`
	AWSRegion          string `json:"aws_region"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	FeedID             string `json:"feed_id"`
	Filename           string `json:"filename"`
	URLType            string `json:"url_type"`
	ReplaceUrl         string `json:"replace_url"`
	// Host scopes the secret to one hostname, or with a "*." prefix to every
	// subdomain of a domain. It limits the name only, not the scheme or port,
	// and isn't checked for S3 credentials. An empty Host allows any host.
	Host string `json:"host"`
}

// MatchFilename finds secrets associated with a DMFR filename.
func (s Secret) MatchFilename(filename string) bool {
	if filename == "" {
		return false
	}
	return path.Base(s.Filename) == filename
}

// MatchFeed finds secrets associated with a DMFR FeedID.
func (s Secret) MatchFeed(feedid string) bool {
	if feedid == "" {
		return false
	}
	return s.FeedID == feedid
}

// MatchHost reports whether the secret may be sent to host, comparing the
// ASCII forms that net/http dials. It returns an error if Host is not a valid
// scope.
func (s Secret) MatchHost(host string) (bool, error) {
	if s.Host == "" {
		return true, nil
	}
	domain, wildcard := strings.CutPrefix(s.Host, "*.")
	scope, err := asciiHost(domain)
	if err != nil || !isHostname(scope) {
		return false, fmt.Errorf("secret host %q is not a hostname or *.domain", s.Host)
	}
	host, err = asciiHost(host)
	if err != nil {
		// net/http can't dial a name with no ASCII form either.
		return false, nil
	}
	if wildcard {
		// net/http never counts an IPv6 address or zone as a subdomain either.
		return !strings.ContainsAny(host, ":%") && strings.HasSuffix(host, "."+scope), nil
	}
	return host == scope, nil
}

// asciiHost returns host lowercased, in the IDNA ASCII form net/http dials.
func asciiHost(host string) (string, error) {
	if strings.IndexFunc(host, func(r rune) bool { return r >= utf8.RuneSelf }) >= 0 {
		var err error
		if host, err = idna.Lookup.ToASCII(host); err != nil {
			return "", err
		}
	}
	return strings.ToLower(host), nil
}

// isHostname reports whether s is dot-separated, non-empty labels of lowercase
// letters, digits, hyphens and underscores.
func isHostname(s string) bool {
	for _, label := range strings.Split(s, ".") {
		if label == "" || strings.Trim(label, "abcdefghijklmnopqrstuvwxyz0123456789-_") != "" {
			return false
		}
	}
	return true
}
