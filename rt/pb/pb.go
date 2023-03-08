//go:generate protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs-realtime.proto=rt/pb  gtfs-realtime.proto

package pb
