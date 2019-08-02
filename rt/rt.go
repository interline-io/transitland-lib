package rt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	pb "github.com/interline-io/gotransit/rt/transit_realtime"
)

func readmsg(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	_ = data
	fmt.Println(len(data))
	msg := pb.FeedMessage{}
	err = proto.Unmarshal(data, &msg)
	if err != nil {
		panic(err)
	}
	z, _ := json.Marshal(msg)
	fmt.Printf("%s\n", z)
	// fmt.Printf("FeedHeader: %#v\n", msg.Header)
	// for _, m := range msg.Entity {
	// 	fmt.Printf("\tFeedMessage: %#v\n", m)
	// 	if m.TripUpdate != nil {
	// 		z, _ := json.Marshal(m.TripUpdate)
	// 		fmt.Printf("\t\tTripUpdate: %s\n", z)
	// 	}
	// 	if m.Vehicle != nil {
	// 		z, _ := json.Marshal(m.Vehicle)
	// 		fmt.Printf("\t\tVehicle: %s\n", z)
	// 	}
	// 	if m.Alert != nil {
	// 		z, _ := json.Marshal(m.Alert)
	// 		fmt.Printf("\t\tAlert: %s\n", z)
	// 	}
	// }
	return nil
}
