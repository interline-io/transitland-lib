// Package rt provides support for GTFS-RealTime. This API is under development and will change.
package rt

import (
	"bytes"
	"context"
	"os"

	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func FlexDecode(data []byte, msg protoreflect.ProtoMessage) error {
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
func ReadURL(ctx context.Context, address string, opts ...request.RequestOption) (*pb.FeedMessage, error) {
	var out bytes.Buffer
	fr, err := request.AuthenticatedRequest(ctx, &out, address, opts...)
	if err != nil {
		return nil, err
	}
	if fr.FetchError != nil {
		return nil, fr.FetchError
	}
	msg := pb.FeedMessage{}
	if err := FlexDecode(out.Bytes(), &msg); err != nil {
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
	if err := FlexDecode(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
