package dmfr

import (
	"path"
	"strings"
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
	// subdomain of a domain. An empty Host allows any host.
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

// MatchHost reports whether the secret may be sent to host.
func (s Secret) MatchHost(host string) bool {
	if s.Host == "" {
		return true
	}
	host = strings.ToLower(host)
	scope := strings.ToLower(s.Host)
	if domain, ok := strings.CutPrefix(scope, "*."); ok {
		return strings.HasSuffix(host, "."+domain)
	}
	return host == scope
}
