package config

// Config is in a separate package to avoid import cycles.

type Config struct {
	Timeout            int
	Port               string
	UseAuth            string
	DBURL              string
	JwtAudience        string
	JwtIssuer          string
	JwtPublicKeyFile   string
	GtfsDir            string
	GtfsS3Bucket       string
	ValidateLargeFiles bool
	DisableImage       bool
	RestPrefix         string
}
