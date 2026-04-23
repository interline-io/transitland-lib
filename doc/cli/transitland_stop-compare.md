## transitland stop-compare

Geometrically compare two GTFS feeds using stop point clouds

### Synopsis

Geometrically compare two GTFS feeds using stop point clouds



```
transitland stop-compare [flags] <feed1> <feed2>
```

### Options

```
      --annd-ratio float     Normalized ANND threshold for 'well matched' stops (0-1) (default 0.02)
      --bbox-iou float       Bounding box IoU threshold for 'same' classification (default 0.75)
      --bbox-overlap float   Bounding box overlap coefficient threshold for subset/superset (default 0.9)
      --boarding-only        Only consider stops with location_type=0 (boarding stops)
  -h, --help                 help for stop-compare
      --json                 Output result as JSON
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

