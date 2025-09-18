//go:generate protoc --plugin=protoc-gen-go=../../protoc-gen-go-wrapper.sh --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs-realtime.proto=rt/pb  gtfs-realtime.proto

package pb
