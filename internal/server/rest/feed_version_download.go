package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

const latestFeedVersionQuery = `
query($feed_onestop_id: String!, $ids: [Int!]) {
	feeds(ids: $ids, where: { onestop_id: $feed_onestop_id }) {
	  license {
		redistribution_allowed
	  }
	  feed_versions(limit: 1) {
		sha1
	  }
	}
  }
`

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func feedDownloadLatestFeedVersionHandler(cfg restConfig, w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["feed_key"]
	gvars := hw{}
	if key == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["ids"] = []int{v}
	} else {
		gvars["feed_onestop_id"] = key
	}
	// Check if we're allowed to redistribute feed and look up latest feed version
	feedResponse, err := makeGraphQLRequest(cfg.srv, latestFeedVersionQuery, gvars)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	found := false
	allowed := false
	json, err := json.Marshal(feedResponse)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
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
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !allowed {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}
	if cfg.GtfsS3Bucket != "" {
		signedURL, err := generatePresignedURLForFeedVersion(cfg.GtfsS3Bucket, fvsha1)
		if err != nil {
			http.Error(w, "failed to create signed url", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Location", signedURL)
		w.WriteHeader(http.StatusFound)
	} else {
		p := filepath.Join(cfg.GtfsDir, fmt.Sprintf("%s.zip", fvsha1))
		http.ServeFile(w, r, p)
	}
}

const feedVersionFileQuery = `
query($feed_version_sha1:String!, $ids: [Int!]) {
	feed_versions(limit:1, ids: $ids, where:{sha1:$feed_version_sha1}) {
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
func fvDownloadHandler(cfg restConfig, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gvars := hw{}
	key := vars["feed_version_key"]
	if key == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["ids"] = []int{v}
	} else {
		gvars["feed_version_sha1"] = key
	}
	fmt.Println("gvars:", gvars)

	// Check if we're allowed to redistribute feed
	checkfv, err := makeGraphQLRequest(cfg.srv, feedVersionFileQuery, gvars)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	// todo: use gjson
	found := false
	allowed := false
	fvsha1 := ""
	if v, ok := checkfv["feed_versions"].([]interface{}); len(v) > 0 && ok {
		found = true
		if v2, ok := v[0].(hw); ok {
			fvsha1 = v2["sha1"].(string)
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
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !allowed {
		http.Error(w, "not authorized", http.StatusUnauthorized)
		return
	}
	if cfg.GtfsS3Bucket != "" {
		signedURL, err := generatePresignedURLForFeedVersion(cfg.GtfsS3Bucket, fvsha1)
		if err != nil {
			http.Error(w, "failed to create signed url", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Location", signedURL)
		w.WriteHeader(http.StatusFound)
	} else {
		p := filepath.Join(cfg.GtfsDir, fmt.Sprintf("%s.zip", fvsha1))
		http.ServeFile(w, r, p)
	}
}

func generatePresignedURLForFeedVersion(s3bucket string, fvHash string) (string, error) {
	// Initialize a session in that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return "", err
	}
	// Create S3 service client
	u, _ := url.Parse(s3bucket)
	bucket := u.Host
	prefix := u.Path
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fmt.Sprintf("%s/%s.zip", prefix, fvHash)),
	})
	return req.Presign(1 * time.Hour)
}
