Feed that conntains stops.txt entities that reference a parent stop that does not contain a valid location_type. 

LAKE_platform_platform references a platform type as parent.
LAKE_platform_boarding_area_invalid references a station type as parent.

Note: a platform referencing another platform or boarding area will also get an InvalidReferenceError.