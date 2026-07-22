//go:build js

package request

import "errors"

// The s3/az stores depend on cloud SDKs (aws-sdk-go-v2, azure-sdk-for-go) that
// don't build for js/wasm (memory-mapped buffers, etc.). They are excluded on
// js/wasm; these stubs keep the scheme switches in GetStore/NewRequest building.
// Returning a Store nil is fine — the call sites assign it to Store/Downloader
// and surface the error.

func NewS3FromUrl(string) (Store, error) {
	return nil, errors.New("s3:// storage is not supported on js/wasm")
}

func NewAzFromUrl(string) (Store, error) {
	return nil, errors.New("az:// storage is not supported on js/wasm")
}
