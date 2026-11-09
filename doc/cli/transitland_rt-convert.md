## transitland rt-convert

Convert GTFS Realtime to JSON

### Synopsis

Convert GTFS Realtime to JSON

Convert GTFS Realtime protocol buffer files to JSON format. Eases inspecting live feeds. Enables processing with JSON-based tools like jq. For vehicle position feeds, you can also convert to GeoJSON (FeatureCollection) or GeoJSONL (one feature per line) formats for visualization or geographic analysis. See https://www.interline.io/blog/geojsonl-extracts/ for more information about GeoJSONL. Note: GeoJSON formats only include vehicle position; trip updates and service alerts can be converted to JSON but not GeoJSON/GeoJSONL.

```
transitland rt-convert [flags] <input pb>
```

### Examples

```
% transitland rt-convert "trips.pb"
% transitland rt-convert --format geojson "vehicle_positions.pb"
% transitland rt-convert --format geojsonl "vehicle_positions.pb"
% transitland rt-convert --format json --out output.json "alerts.pb"
% transitland rt-convert --format geojson --out mbta_vehicles.geojson "https://cdn.mbta.com/realtime/VehiclePositions.pb"
% transitland rt-convert --format geojson "https://developer.trimet.org/ws/gtfs/VehiclePositions"
% transitland rt-convert --format json "https://developer.trimet.org/ws/V1/TripUpdate"
```

### Options

```
  -f, --format string        Output format: json, geojson, geojsonl (geojson formats only convert vehicle position entities) (default "json")
      --header stringArray   Request header as 'Name: value', replacing any default of the same name; may be repeated
  -h, --help                 help for rt-convert
  -o, --out string           Write output to file; defaults to stdout
      --url-type string      DMFR feed.urls key for the request; selects request headers (default "realtime")
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

