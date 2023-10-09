// Package rt provides support for GTFS-RealTime. This API is under development and will change.
package rt

import (
	"io/ioutil"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl/request"
	"google.golang.org/protobuf/proto"
)

// ReadURL opens a message from a url.
func ReadURL(address string, opts ...request.RequestOption) (*pb.FeedMessage, error) {
	fr, err := request.AuthenticatedRequest(address, opts...)
	if err != nil {
		return nil, err
	}
	data := fr.Data
	msg := pb.FeedMessage{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
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
