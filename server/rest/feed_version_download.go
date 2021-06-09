package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

const latestFeedVersionQuery = `
query($feed_onestop_id: String!) {
	feeds(where: { onestop_id: $feed_onestop_id }) {
	  license {
		redistribution_allowed
	  }
	  feed_versions(limit: 1) {
		sha1
	  }
	}
  }
`

const feedVersionFileQuery = `
query($feed_version_sha1:String!) {
	feed_versions(limit:1, where:{sha1:$feed_version_sha1}) {
	  sha1
	  feed {
		license {
			redistribution_allowed
		}
	  }
	}
  }
`

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func feedDownloadLatestFeedVersionHandler(cfg restConfig, w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	gvars := hw{}
	if key == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["id"] = v
	} else {
		gvars["feed_onestop_id"] = key
	}
	// Check if we're allowed to redistribute feed and look up latest feed version
	feedResponse, err := makeGraphQLRequest(cfg.srv, latestFeedVersionQuery, gvars)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	found := false
	allowed := false
	json, err := json.Marshal(feedResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if gjson.Get(string(json), "feeds.0.feed_versions.0.sha1").Exists() {
		found = true
	}
	if gjson.Get(string(json), "feeds.0.license.redistribution_allowed").String() != "no" {
		allowed = true
	}
	fvsha1 := gjson.Get(string(json), "feeds.0.feed_versions.0.sha1").String()
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !allowed {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	signedURL, err := generatePresignedURLForFeedVersion(fvsha1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Location", signedURL)
	w.WriteHeader(http.StatusFound)
}

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func fvDownloadHandler(cfg restConfig, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fvsha1 := vars["key"]
	// Check if we're allowed to redistribute feed
	checkfv, err := makeGraphQLRequest(cfg.srv, feedVersionFileQuery, hw{"feed_version_sha1": fvsha1})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	found := false
	allowed := false
	if v, ok := checkfv["feed_versions"].([]interface{}); len(v) > 0 && ok {
		found = true
		if v2, ok := v[0].(hw); ok {
			if v3, ok := v2["feed"].(hw); ok {
				if v4, ok := v3["license"].(hw); ok {
					if v4["redistribution_allowed"] != "no" {
						allowed = true
					}
				}
			}
		}
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !allowed {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	//
	signedURL, err := generatePresignedURLForFeedVersion(fvsha1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Location", signedURL)
	w.WriteHeader(http.StatusFound)
}

func generatePresignedURLForFeedVersion(fvHash string) (string, error) {
	// Initialize a session in that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return "", err
	}
	// Create S3 service client
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("transitland-gtfs"),
		Key:    aws.String(fmt.Sprintf("datastore-uploads/feed_version/%s.zip", fvHash)),
	})
	return req.Presign(1 * time.Hour)
}
