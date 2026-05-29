package adminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/fga"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fgaServer backs adminapi.NewServer with a real OpenFGA client for the
// permission-mutation routes. It embeds the (nil) Server interfaces and
// overrides only Add/RemovePermission, writing the exact tuple the handler
// builds to OpenFGA — the same FGA write that Checker.AddPermission performs,
// minus the acting-user authorization checks (Check/entityExists need a
// Postgres DB and are covered in the azchecker tests).
//
// This makes the real OpenFGA authorization model the validator: a subject
// that drops its ref relation (e.g. bare "tenant:1" instead of
// "tenant:1#member") is not an allowed type restriction, so OpenFGA rejects
// the write, AddPermission returns an error, and the handler responds 500 —
// which is precisely the bug this guards against.
type fgaServer struct {
	authz.AdminManager
	authz.EntityProvider
	fga       *fga.FGAClient
	lastWrite authz.TupleKey
}

func (s *fgaServer) tupleFor(obj authz.ObjectRef, subject authz.EntityKey, relation authz.Relation) authz.TupleKey {
	// Mirrors the object key construction in Checker.AddPermission.
	return authz.TupleKey{
		Subject:  subject,
		Object:   authz.NewEntityKey(obj.Type, strconv.FormatInt(obj.ID, 10)),
		Relation: relation,
	}
}

func (s *fgaServer) AddPermission(ctx context.Context, obj authz.ObjectRef, subject authz.EntityKey, relation authz.Relation) error {
	tk := s.tupleFor(obj, subject, relation)
	s.lastWrite = tk
	return s.fga.WriteTuple(ctx, tk)
}

func (s *fgaServer) RemovePermission(ctx context.Context, obj authz.ObjectRef, subject authz.EntityKey, relation authz.Relation) error {
	tk := s.tupleFor(obj, subject, relation)
	s.lastWrite = tk
	return s.fga.DeleteTuple(ctx, tk)
}

// newModeledFGAClient creates a fresh store on the shared in-memory OpenFGA
// server and loads the production authorization model (tls.json).
func newModeledFGAClient(t *testing.T, url string) *fga.FGAClient {
	t.Helper()
	c, err := fga.NewFGAClient(url, "", "")
	require.NoError(t, err)
	_, err = c.CreateStore(context.Background(), "adminapi-test")
	require.NoError(t, err)
	_, err = c.CreateModel(context.Background(), testdata.Path("server/authz/tls.json"))
	require.NoError(t, err)
	return c
}

// TestPermissionHandlersWriteValidTuples drives the real HTTP handlers against
// an in-memory OpenFGA server and asserts that the tuple each handler builds
// from the request body is accepted by the production model. Before the fix
// the handlers dropped ref_relation, so the group/feed-version cases wrote a
// bare object subject that the model rejects (HTTP 500).
func TestPermissionHandlersWriteValidTuples(t *testing.T) {
	fgaURL := testutil.FGAServer(t)

	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		wantSubject string // EntityKey the handler is expected to build
	}{
		{
			name:        "add tenant#member as group viewer",
			method:      http.MethodPost,
			path:        "/groups/46/permissions",
			body:        `{"id":"1","type":"tenant","ref_relation":"member","relation":"viewer"}`,
			wantSubject: "tenant:1#member",
		},
		{
			name:        "add org#viewer as feed version editor",
			method:      http.MethodPost,
			path:        "/feed_versions/5/permissions",
			body:        `{"id":"39","type":"org","ref_relation":"viewer","relation":"editor"}`,
			wantSubject: "org:39#viewer",
		},
		{
			name:        "add user (no ref relation) as tenant member",
			method:      http.MethodPost,
			path:        "/tenants/1/permissions",
			body:        `{"id":"auth0|abc","type":"user","relation":"member"}`,
			wantSubject: "user:auth0|abc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &fgaServer{fga: newModeledFGAClient(t, fgaURL)}
			h, err := NewServer(srv)
			require.NoError(t, err)

			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			// 200 means OpenFGA accepted the tuple the handler built.
			require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
			assert.Equal(t, tc.wantSubject, srv.lastWrite.Subject.String())
		})
	}
}

// TestModelRejectsBareUsersetSubject documents, against the real model, why the
// fix matters: a group viewer subject must be a userset (tenant#member), not a
// bare object (tenant:1, the pre-fix output of dropping ref_relation).
func TestModelRejectsBareUsersetSubject(t *testing.T) {
	c := newModeledFGAClient(t, testutil.FGAServer(t))
	ctx := context.Background()
	group := authz.NewEntityKey(authz.GroupType, "46")

	bare := authz.TupleKey{
		Subject:  authz.NewEntityKey(authz.TenantType, "1"),
		Object:   group,
		Relation: authz.ViewerRelation,
	}
	assert.Error(t, c.WriteTuple(ctx, bare),
		"model should reject bare tenant:1 as a group viewer (this is the dropped-ref_relation bug)")

	userset := authz.TupleKey{
		Subject:  authz.NewEntityKey(authz.TenantType, "1").WithRefRel(authz.MemberRelation),
		Object:   group,
		Relation: authz.ViewerRelation,
	}
	assert.NoError(t, c.WriteTuple(ctx, userset),
		"model should accept tenant:1#member as a group viewer")
}
