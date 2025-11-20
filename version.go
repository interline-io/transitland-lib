// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

// Read version from compiled in git details
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
	tagPrefix := "main.tag="
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			ret.Commit = kv.Value
		case "vcs.time":
			ret.CommitTime = kv.Value
		case "-ldflags":
			for _, ss := range strings.Split(kv.Value, " ") {
				if strings.HasPrefix(ss, tagPrefix) {
					ret.Tag = strings.TrimPrefix(ss, tagPrefix)
				}
			}
		}
	}
	return ret
}

// GTFSVERSION is the commit for the spec reference.md file.
var GTFSVERSION = "4f0166688c980ee30e6457035616bf96aa21d91f" // NOTE: some intervening changes have not yet been added

// GTFSRTVERSION is the commit for the gtfs-realtime.proto file.
var GTFSRTVERSION = "7b9f229dfa0b539c3fcf461986638890024feb06"
