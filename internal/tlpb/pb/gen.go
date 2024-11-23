//go:generate protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs.proto=tlpb/pb -I .. ../gtfs.proto

package pb
