package model

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// CensusCursor encodes pagination state for census values using composite key (geoid, table_id)
type CensusCursor struct {
	Geoid   string
	TableID int
	Valid   bool
}

// NewCensusCursor creates a new census cursor
func NewCensusCursor(geoid string, tableID int) CensusCursor {
	return CensusCursor{
		Geoid:   geoid,
		TableID: tableID,
		Valid:   true,
	}
}

// Encode returns a base64-encoded cursor string
func (c *CensusCursor) Encode() string {
	if !c.Valid {
		return ""
	}
	raw := fmt.Sprintf("%s,%d", c.Geoid, c.TableID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCensusCursor decodes a cursor string
func DecodeCensusCursor(cursor string) (CensusCursor, error) {
	if cursor == "" {
		return CensusCursor{}, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return CensusCursor{}, fmt.Errorf("invalid cursor format")
	}

	parts := strings.SplitN(string(decoded), ",", 2)
	if len(parts) != 2 {
		return CensusCursor{}, fmt.Errorf("invalid cursor structure")
	}

	tableID, err := strconv.Atoi(parts[1])
	if err != nil {
		return CensusCursor{}, fmt.Errorf("invalid table_id in cursor")
	}

	return CensusCursor{
		Geoid:   parts[0],
		TableID: tableID,
		Valid:   true,
	}, nil
}

// CensusValueConnection represents a Relay-style connection for census values
type CensusValueConnection struct {
	Edges    []*CensusValueEdge
	PageInfo *PageInfo
}

// CensusValueEdge represents an edge in the census values connection
type CensusValueEdge struct {
	Node   *CensusValue
	Cursor string
}

// PageInfo contains pagination metadata
type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     *string
	EndCursor       *string
}
