// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	"runtime/debug"
	"strings"
)

var Version VersionInfo

func init() {
	Version = getVersion()
}

type VersionInfo struct {
	Tag        string
	Commit     string
	CommitTime string
}

func getVersion() VersionInfo {
	ret := VersionInfo{}
	info, _ := debug.ReadBuildInfo()
	if info != nil {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			ret.Tag = info.Main.Version
		}
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				ret.Commit = kv.Value
			case "vcs.time":
				ret.CommitTime = kv.Value
			case "-ldflags":
				if idx := strings.Index(kv.Value, "-X main.tag="); idx != -1 {
					start := idx + len("-X main.tag=")
					end := start
					for end < len(kv.Value) && kv.Value[end] != ' ' {
						end++
					}
					if ret.Tag == "" {
						ret.Tag = kv.Value[start:end]
					}
				}
			}
		}
	}
	return ret
}

// GTFSVERSION is the commit for the spec reference.md file.
var GTFSVERSION = "11a49075c1f50d0130b934833b7eeb3fe518961c"

// GTFSRTVERSION is the commit for the gtfs-realtime.proto file.
var GTFSRTVERSION = "7b9f229dfa0b539c3fcf461986638890024feb06"
