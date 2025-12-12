# flex-trip-create-shapes

Test feed for verifying that `CreateMissingShapes` only generates shapes for fixed-route trips where all stop_times have a `stop_id`. Contains three trips: a regular fixed-route trip (should get a generated shape), a flex trip using `location_id` (should not get a shape), and a trip with an existing shape (should keep its original shape).
