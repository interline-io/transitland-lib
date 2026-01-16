BEGIN;

-- Migration: Add direction_id, sequence_idx, and way_id to tl_segment_patterns
-- This allows reconstruction of the original map-matched output for each shape

-- Add direction_id column
ALTER TABLE tl_segment_patterns 
ADD COLUMN IF NOT EXISTS direction_id integer NOT NULL DEFAULT 0;

-- Add sequence_idx column to track order within a shape
ALTER TABLE tl_segment_patterns 
ADD COLUMN IF NOT EXISTS sequence_idx integer NOT NULL DEFAULT 0;

-- Add way_id column (denormalized from segment for future architecture)
ALTER TABLE tl_segment_patterns 
ADD COLUMN IF NOT EXISTS way_id bigint;

-- Populate way_id from existing segments data (cast from text to bigint)
UPDATE tl_segment_patterns sp
SET way_id = s.way_id::bigint
FROM tl_segments s
WHERE sp.segment_id = s.id;

-- Make way_id non-null after populating
ALTER TABLE tl_segment_patterns 
ALTER COLUMN way_id SET NOT NULL;

-- Drop old unique constraint that doesn't include sequence_idx
-- Old: (segment_id, route_id, shape_id, stop_pattern_id)
-- New: include sequence_idx since same segment can appear multiple times in a shape
DROP INDEX IF EXISTS tl_segment_patterns_segment_id_route_id_shape_id_stop_patte_idx;

-- Create new unique constraint including sequence_idx
CREATE UNIQUE INDEX IF NOT EXISTS tl_segment_patterns_segment_route_shape_pattern_seq_idx 
ON tl_segment_patterns (segment_id, route_id, shape_id, stop_pattern_id, sequence_idx);

-- Create index for efficient shape reconstruction queries
CREATE INDEX IF NOT EXISTS tl_segment_patterns_shape_sequence_idx 
ON tl_segment_patterns (feed_version_id, shape_id, sequence_idx);

-- Create index for direction-based queries
CREATE INDEX IF NOT EXISTS tl_segment_patterns_direction_idx 
ON tl_segment_patterns (feed_version_id, route_id, direction_id);

COMMIT;