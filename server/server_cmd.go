package server

import (
	"flag"
	"log"
	"os"

	"github.com/interline-io/transitland-lib/server/config"
)

type Command struct {
	config.Config
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("sync", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: server")
		fl.PrintDefaults()
	}
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.IntVar(&cmd.Timeout, "timeout", 60, "")
	fl.StringVar(&cmd.Port, "port", "8080", "")
	fl.StringVar(&cmd.JwtAudience, "jwt-audience", "", "JWT Audience")
	fl.StringVar(&cmd.JwtIssuer, "jwt-issuer", "", "JWT Issuer")
	fl.StringVar(&cmd.JwtPublicKeyFile, "jwt-public-key-file", "", "Path to JWT public key file")
	fl.StringVar(&cmd.UseAuth, "auth", "", "")
	fl.StringVar(&cmd.GtfsDir, "gtfsdir", "", "Directory to store GTFS files")
	fl.StringVar(&cmd.GtfsS3Bucket, "s3", "", "S3 bucket for GTFS files")
	fl.BoolVar(&cmd.ValidateLargeFiles, "validate-large-files", false, "Allow validation of large files")
	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

func (cmd *Command) Run(args []string) error {
	// TODO: fix interface
	if err := cmd.Parse(args); err != nil {
		panic(err)
	}
	return Serve(cmd.Config)
}
