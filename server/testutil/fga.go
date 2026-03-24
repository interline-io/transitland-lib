package testutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	openfga "github.com/openfga/openfga/pkg/server"
	"github.com/openfga/openfga/pkg/storage/memory"
)

// FGAServer starts an in-process OpenFGA server with in-memory storage
// and returns its HTTP endpoint URL. The server is automatically shut down
// when the test completes.
//
// If TL_TEST_FGA_ENDPOINT is set, it returns that URL instead (for CI
// service containers or external servers).
func FGAServer(t testing.TB) string {
	t.Helper()

	// Use external server if configured
	if endpoint := os.Getenv("TL_TEST_FGA_ENDPOINT"); endpoint != "" {
		return endpoint
	}

	// Create in-memory datastore and server
	ds := memory.New()
	srv, err := openfga.NewServerWithOpts(openfga.WithDatastore(ds))
	if err != nil {
		t.Fatalf("could not create OpenFGA server: %v", err)
	}

	// Register gRPC-gateway HTTP handler
	mux := runtime.NewServeMux()
	if err := openfgav1.RegisterOpenFGAServiceHandlerServer(context.Background(), mux, srv); err != nil {
		t.Fatalf("could not register OpenFGA HTTP handler: %v", err)
	}

	// Listen on a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	endpoint := fmt.Sprintf("http://localhost:%d", port)

	// Start HTTP server
	httpSrv := &http.Server{Handler: mux}
	go httpSrv.Serve(l)

	// Cleanup on test completion
	t.Cleanup(func() {
		httpSrv.Close()
		srv.Close()
	})

	return endpoint
}
