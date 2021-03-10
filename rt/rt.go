// Package rt provides support for GTFS-RealTime. This API is under development and will change.
package rt

import (
	"io/ioutil"
	"net/url"

	"github.com/interline-io/transitland-lib/internal/download"
	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/proto"
)

// Read opens a message from a file or url.
func Read(address string) (*pb.FeedMessage, error) {
	if u, err := url.Parse(address); err == nil {
		if u.Scheme == "http" || u.Scheme == "https" {
			return ReadURL(address)
		}
	}
	return ReadFile(address)
}

// ReadURL opens a message from a url.
func ReadURL(address string) (*pb.FeedMessage, error) {
	td := download.TemporaryDownload{URL: address}
	if err := td.Open(); err != nil {
		return nil, err
	}
	defer td.Close()
	return ReadFile(td.File.Name())
}

// ReadFile opens a message from a file.
func ReadFile(filename string) (*pb.FeedMessage, error) {
	msg := pb.FeedMessage{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return &msg, err
	}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return &msg, err
	}
	return &msg, nil
}
