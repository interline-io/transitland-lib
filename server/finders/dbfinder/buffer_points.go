package dbfinder

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/server/model"
)

// bufferPoints are the points of a stop buffer.
type bufferPoints []*model.BufferPoint

// ToSql selects the points as the columns id, label and geometry.
func (p bufferPoints) ToSql() (string, []any, error) {
	// No points are an empty array, since json_to_recordset rejects null.
	if p == nil {
		p = bufferPoints{}
	}
	// The points reach the query as JSON objects in one parameter, which pairs
	// each value with its column. A key left out reads as NULL.
	data, err := json.Marshal(p)
	if err != nil {
		return "", nil, fmt.Errorf("encoding buffer points: %w", err)
	}
	return "SELECT p.id, p.label, ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326)::geography AS geometry FROM json_to_recordset(?::json) AS p(id bigint, label text, lon float8, lat float8)", []any{string(data)}, nil
}

// sharesOneBuffer reports whether every point has the same id and label, and
// so falls in one buffer.
func sharesOneBuffer(points []*model.BufferPoint) bool {
	for _, p := range points {
		if !equalPtr(p.ID, points[0].ID) || !equalPtr(p.Label, points[0].Label) {
			return false
		}
	}
	return true
}

// equalPtr reports whether two pointers are both nil or point to equal values.
func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
