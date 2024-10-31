package dmfr

import (
	"path"
)

// Secret .
type Secret struct {
	Key                string `json:"key"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	FeedID             string `json:"feed_id"`
	Filename           string `json:"filename"`
	URLType            string `json:"url_type"`
	ReplaceUrl         string `json:"replace_url"`
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
