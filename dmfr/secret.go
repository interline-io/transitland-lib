package dmfr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

// Secrets loads a JSON file of secrets
type Secrets []Secret

// Load secrets from a file.
func (s *Secrets) Load(filename string) error {
	bv, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	json.Unmarshal(bv, s)
	return nil
}

// MatchFilename finds secrets associated with a DMFR filename.
func (s Secrets) MatchFilename(filename string) (Secret, error) {
	if len(filename) == 0 {
		return Secret{}, errors.New("no filename provided")
	}
	found := Secret{}
	count := 0
	for _, secret := range s {
		if secret.Filename == filename {
			count++
			found = secret
		}
	}
	if count == 0 {
		return Secret{}, errors.New("no results")
	} else if count > 1 {
		return Secret{}, fmt.Errorf("ambiguous results; %d matches", count)
	}
	return found, nil
}

// MatchFeed finds secrets associated with a DMFR FeedID.
func (s Secrets) MatchFeed(feedid string) (Secret, error) {
	if len(feedid) == 0 {
		return Secret{}, errors.New("no feedid provided")
	}
	found := Secret{}
	count := 0
	for _, secret := range s {
		if secret.FeedID == feedid {
			count++
			found = secret
		}
	}
	if count == 0 {
		return Secret{}, errors.New("no results")
	} else if count > 1 {
		return Secret{}, fmt.Errorf("ambiguous results; %d matches", count)
	}
	return found, nil
}

// Secret .
type Secret struct {
	Key      string `json:"key"`
	Username string `json:"username"`
	Password string `json:"password"`
	FeedID   string `json:"feed_id"`
	Filename string `json:"filename"`
}
