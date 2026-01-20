package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCensusCursor(t *testing.T) {
	cursor := NewCensusCursor("0500000US06001", 42)
	assert.True(t, cursor.Valid)
	assert.Equal(t, "0500000US06001", cursor.Geoid)
	assert.Equal(t, 42, cursor.TableID)
}

func TestCensusCursor_Encode(t *testing.T) {
	testCases := []struct {
		name     string
		cursor   CensusCursor
		expected string
	}{
		{
			name:     "valid cursor encodes to base64",
			cursor:   NewCensusCursor("0500000US06001", 42),
			expected: "MDUwMDAwMFVTMDYwMDEsNDI",
		},
		{
			name:     "invalid cursor returns empty string",
			cursor:   CensusCursor{Valid: false},
			expected: "",
		},
		{
			name:     "cursor with zero table ID",
			cursor:   NewCensusCursor("test", 0),
			expected: "dGVzdCww",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cursor.Encode()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDecodeCensusCursor(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    CensusCursor
		expectError bool
		errorMsg    string
	}{
		{
			name:     "empty string returns invalid cursor",
			input:    "",
			expected: CensusCursor{Valid: false},
		},
		{
			name:     "valid encoded cursor",
			input:    "MDUwMDAwMFVTMDYwMDEsNDI",
			expected: NewCensusCursor("0500000US06001", 42),
		},
		{
			name:     "cursor with zero table ID",
			input:    "dGVzdCww",
			expected: NewCensusCursor("test", 0),
		},
		{
			name:        "invalid base64 encoding",
			input:       "not-valid-base64!!!",
			expectError: true,
			errorMsg:    "invalid cursor format",
		},
		{
			name:        "missing comma separator",
			input:       "bm9jb21tYQ", // "nocomma" in base64
			expectError: true,
			errorMsg:    "invalid cursor structure",
		},
		{
			name:        "non-numeric table ID",
			input:       "dGVzdCxhYmM", // "test,abc" in base64
			expectError: true,
			errorMsg:    "invalid table_id in cursor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecodeCensusCursor(tc.input)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected.Valid, result.Valid)
				assert.Equal(t, tc.expected.Geoid, result.Geoid)
				assert.Equal(t, tc.expected.TableID, result.TableID)
			}
		})
	}
}

func TestCensusCursor_RoundTrip(t *testing.T) {
	testCases := []struct {
		name    string
		geoid   string
		tableID int
	}{
		{
			name:    "county geoid",
			geoid:   "0500000US06001",
			tableID: 1,
		},
		{
			name:    "tract geoid",
			geoid:   "1400000US06001403000",
			tableID: 99,
		},
		{
			name:    "large table ID",
			geoid:   "test",
			tableID: 999999,
		},
		{
			name:    "special characters in geoid",
			geoid:   "ntd:00001",
			tableID: 5,
		},
		{
			name:    "empty geoid",
			geoid:   "",
			tableID: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := NewCensusCursor(tc.geoid, tc.tableID)
			encoded := original.Encode()
			decoded, err := DecodeCensusCursor(encoded)

			assert.NoError(t, err)
			assert.True(t, decoded.Valid)
			assert.Equal(t, original.Geoid, decoded.Geoid)
			assert.Equal(t, original.TableID, decoded.TableID)
		})
	}
}

func TestCensusCursor_EdgeCases(t *testing.T) {
	t.Run("geoid containing comma fails to decode", func(t *testing.T) {
		// Known limitation: geoids containing commas will fail to decode
		// because the cursor format uses comma as separator (geoid,tableID)
		// and SplitN splits from the left. This is acceptable because
		// real geoids (e.g., "0500000US06001", "1400000US06001403000", "ntd:00001")
		// do not contain commas.
		original := NewCensusCursor("geoid,with,commas", 10)
		encoded := original.Encode()
		_, err := DecodeCensusCursor(encoded)

		// This should fail because "with,commas,10" is not a valid integer
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid table_id")
	})

	t.Run("negative table ID", func(t *testing.T) {
		original := NewCensusCursor("test", -1)
		encoded := original.Encode()
		decoded, err := DecodeCensusCursor(encoded)

		assert.NoError(t, err)
		assert.Equal(t, -1, decoded.TableID)
	})
}
