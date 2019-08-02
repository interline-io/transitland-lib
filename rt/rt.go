package rt

import (
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	pb "github.com/interline-io/gotransit/rt/transit_realtime"
)

func readmsg(filename string) (pb.FeedMessage, error) {
	msg := pb.FeedMessage{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return msg, err
	}
	if err := proto.Unmarshal(data, &msg); err != nil {
		return msg, err
	}
	return msg, nil
}
