package rest

import (
	"context"
	"fmt"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/multireader"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext/filters"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
)

// FeedVersionExporter handles exporting feed versions with transformations
type FeedVersionExporter struct {
	cfg *model.Config
}

// NewFeedVersionExporter creates a new feed version exporter
func NewFeedVersionExporter(cfg *model.Config) *FeedVersionExporter {
	return &FeedVersionExporter{cfg: cfg}
}

// Export performs the feed version export with optional transformations
func (e *FeedVersionExporter) Export(ctx context.Context, fvids []int, transforms *ExportTransforms, writer adapters.Writer) (*copier.Result, error) {
	// Create database readers for each feed version
	var readers []adapters.Reader
	dbx := e.cfg.Finder.DBX()

	for _, fvid := range fvids {
		reader := &tldb.Reader{
			Adapter:        postgres.NewPostgresAdapterFromDBX(dbx),
			PageSize:       1_000,
			FeedVersionIDs: []int{fvid},
		}
		if err := reader.Open(); err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("failed to open feed version reader")
			return nil, fmt.Errorf("failed to open feed version reader for %d: %s", fvid, err.Error())
		}
		defer reader.Close()
		readers = append(readers, reader)
	}

	// Use multireader if multiple feed versions, otherwise use single reader
	var reader adapters.Reader
	if len(readers) == 1 {
		reader = readers[0]
	} else {
		reader = multireader.NewReader(readers...)
		if err := reader.Open(); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to open multireader")
			return nil, fmt.Errorf("failed to open multireader: %s", err.Error())
		}
		defer reader.Close()
	}

	// Configure copier options with transformations
	opts := copier.Options{
		AllowEntityErrors:    true,
		AllowReferenceErrors: false,
		ErrorLimit:           1000,
		Quiet:                true,
	}

	// Apply transformations
	if transforms != nil {
		if err := e.applyTransforms(&opts, transforms, fvids); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to apply transforms")
			return nil, fmt.Errorf("failed to apply transforms: %s", err.Error())
		}
	}

	// Perform the copy operation (streaming to ZIP)
	result, err := copier.CopyWithOptions(ctx, reader, writer, opts)
	if err != nil {
		// Can't write error to response as headers are already sent
		log.For(ctx).Error().Err(err).Msg("export failed")
		return nil, fmt.Errorf("export failed: %s", err.Error())
	}

	return result, nil
}

// applyTransforms configures copier options based on transform request
func (e *FeedVersionExporter) applyTransforms(opts *copier.Options, transforms *ExportTransforms, fvids []int) error {
	// ID prefix/namespacing
	if transforms.Prefix != "" {
		prefixFilter, err := filters.NewPrefixFilter()
		if err != nil {
			return fmt.Errorf("failed to create prefix filter: %w", err)
		}

		// Set prefix for each feed version
		for _, fvid := range fvids {
			prefixFilter.SetPrefix(fvid, transforms.Prefix)
		}

		// Configure which files to prefix
		if len(transforms.PrefixFiles) > 0 {
			for _, file := range transforms.PrefixFiles {
				prefixFilter.PrefixFile(file)
			}
		}

		opts.AddExtension(prefixFilter)
	}

	// Normalize timezones
	if transforms.NormalizeTimezones {
		opts.NormalizeTimezones = true
	}

	// Simplify shapes
	if transforms.SimplifyShapes != nil && *transforms.SimplifyShapes > 0 {
		opts.SimplifyShapes = *transforms.SimplifyShapes
	}

	// Use basic route types
	if transforms.UseBasicRouteTypes {
		opts.UseBasicRouteTypes = true
	}

	// Set specific values
	if len(transforms.SetValues) > 0 {
		setterFilter := extract.NewSetterFilter()
		for key, value := range transforms.SetValues {
			// Parse key format: "filename.entity_id.field"
			parts := strings.SplitN(key, ".", 3)
			if len(parts) != 3 {
				return fmt.Errorf("invalid set_values key format: %s (expected: filename.entity_id.field)", key)
			}
			setterFilter.AddValue(parts[0], parts[1], parts[2], value)
		}
		opts.AddExtension(setterFilter)
	}

	return nil
}
