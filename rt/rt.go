// Package rt provides support for GTFS-RealTime. This API is under development and will change.
package rt

import (
	"os"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl/request"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func flexDecode(data []byte, msg protoreflect.ProtoMessage) error {
	err := proto.Unmarshal(data, msg)
	if err == nil {
		return nil
	}
	if len(data) > 0 && data[0] == '{' {
		// Try again, still using err
		err = protojson.Unmarshal(data, msg)
	}
	return err
}

// ReadURL opens a message from a url.
func ReadURL(address string, opts ...request.RequestOption) (*pb.FeedMessage, error) {
	fr, err := request.AuthenticatedRequest(address, opts...)
	if err != nil {
		return nil, err
	}
	msg := pb.FeedMessage{}
	data := fr.Data
	if err := flexDecode(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ReadFile opens a message from a file.
func ReadFile(filename string) (*pb.FeedMessage, error) {
	msg := pb.FeedMessage{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return &msg, err
	}
	if err := flexDecode(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
