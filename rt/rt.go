// Package rt provides support for GTFS-RealTime. This API is under development and will change.
package rt

import (
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/interline-io/transitland-lib/rt/pb"
)

func readmsg(filename string) (*pb.FeedMessage, error) {
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
