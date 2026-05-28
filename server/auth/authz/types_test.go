package authz

import (
	"encoding/json"
	"testing"
)

func TestObjectType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ObjectType
		wantErr bool
	}{
		{"number user", `5`, ObjectType_user, false},
		{"number feed", `3`, ObjectType_feed, false},
		{"number empty", `0`, ObjectType_empty_object, false},
		{"string user", `"user"`, ObjectType_user, false},
		{"string feed_version", `"feed_version"`, ObjectType_feed_version, false},
		{"string group alias for org", `"group"`, ObjectType_org, false},
		{"unknown number", `999`, 0, true},
		{"unknown string", `"frobnicator"`, 0, true},
		{"invalid json", `{}`, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got ObjectType
			err := json.Unmarshal([]byte(tc.input), &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestRelation_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Relation
		wantErr bool
	}{
		{"number viewer", `4`, Relation_viewer, false},
		{"number admin", `1`, Relation_admin, false},
		{"number empty", `0`, Relation_empty_relation, false},
		{"string viewer", `"viewer"`, Relation_viewer, false},
		{"string manager", `"manager"`, Relation_manager, false},
		{"unknown number", `99`, 0, true},
		{"unknown string", `"overlord"`, 0, true},
		{"invalid json", `[]`, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Relation
			err := json.Unmarshal([]byte(tc.input), &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestEntityRelation_UnmarshalJSON_IntegerEnums(t *testing.T) {
	// Reproduces the admin permissions request body shape sent by the
	// tlv2-admin frontend. The admin-api migration guide documents that
	// integer enum values are accepted.
	body := `{"id":"auth0|6a18684b0cd1dfbdd0beb98c","type":5,"relation":4}`
	var er EntityRelation
	if err := json.Unmarshal([]byte(body), &er); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if er.Type != ObjectType_user {
		t.Errorf("type = %v, want user", er.Type)
	}
	if er.Relation != Relation_viewer {
		t.Errorf("relation = %v, want viewer", er.Relation)
	}
	if er.Id != "auth0|6a18684b0cd1dfbdd0beb98c" {
		t.Errorf("id = %q", er.Id)
	}
}

func TestEntityRelation_UnmarshalJSON_StringEnums(t *testing.T) {
	body := `{"id":"auth0|x","type":"user","relation":"viewer","ref_relation":"member"}`
	var er EntityRelation
	if err := json.Unmarshal([]byte(body), &er); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if er.Type != ObjectType_user || er.Relation != Relation_viewer || er.RefRelation != Relation_member {
		t.Errorf("got %+v", er)
	}
}
