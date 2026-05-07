package stopcompare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/spf13/pflag"
)

// Command is the CLI entry point for stop-compare.
type Command struct {
	ANNDRatio         float64
	BboxIoU           float64
	BboxOverlap       float64
	BoardingStopsOnly bool
	OutputJSON        bool
	readerPathA       string
	readerPathB       string
}

func (cmd *Command) HelpDesc() (string, string) {
	return "Geometrically compare two GTFS feeds using stop point clouds", ""
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <feed1> <feed2>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.Float64Var(&cmd.ANNDRatio, "annd-ratio", 0.02, "Normalized ANND threshold for 'well matched' stops (0-1)")
	fl.Float64Var(&cmd.BboxIoU, "bbox-iou", 0.75, "Bounding box IoU threshold for 'same' classification")
	fl.Float64Var(&cmd.BboxOverlap, "bbox-overlap", 0.90, "Bounding box overlap coefficient threshold for subset/superset")
	fl.BoolVar(&cmd.BoardingStopsOnly, "boarding-only", false, "Only consider stops with location_type=0 (boarding stops)")
	fl.BoolVar(&cmd.OutputJSON, "json", false, "Output result as JSON")
}

func (cmd *Command) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires two input feeds")
	}
	cmd.readerPathA = fl.Arg(0)
	cmd.readerPathB = fl.Arg(1)
	return nil
}

func (cmd *Command) Run(_ context.Context) error {
	readerA, err := tlcsv.NewReader(cmd.readerPathA)
	if err != nil {
		return fmt.Errorf("opening feed A: %w", err)
	}
	if err := readerA.Open(); err != nil {
		return fmt.Errorf("opening feed A: %w", err)
	}
	defer readerA.Close()

	readerB, err := tlcsv.NewReader(cmd.readerPathB)
	if err != nil {
		return fmt.Errorf("opening feed B: %w", err)
	}
	if err := readerB.Open(); err != nil {
		return fmt.Errorf("opening feed B: %w", err)
	}
	defer readerB.Close()

	opts := Options{
		ANNDRatioThreshold:   cmd.ANNDRatio,
		BboxIoUThreshold:     cmd.BboxIoU,
		BboxOverlapThreshold: cmd.BboxOverlap,
		BoardingStopsOnly:    cmd.BoardingStopsOnly,
	}

	result, err := CompareReaders(cmd.readerPathA, readerA, cmd.readerPathB, readerB, opts)
	if err != nil {
		return err
	}

	if cmd.OutputJSON {
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("Feed A: %s (%d stops)\n", result.FeedA, result.StopCountA)
	fmt.Printf("Feed B: %s (%d stops)\n\n", result.FeedB, result.StopCountB)
	fmt.Printf("Coverage (bounding box):\n")
	fmt.Printf("  IoU:                 %.4f\n", result.Bbox.IoU)
	fmt.Printf("  Overlap coefficient: %.4f\n\n", result.Bbox.OverlapCoefficient)
	fmt.Printf("Point matching:\n")
	fmt.Printf("  A→B: mean=%.0fm, median=%.0fm, p90=%.0fm, normalized ANND=%.4f\n",
		result.AtoB.MeanDistMeters, result.AtoB.MedianDistMeters, result.AtoB.P90DistMeters, result.AtoB.NormalizedANND)
	fmt.Printf("  B→A: mean=%.0fm, median=%.0fm, p90=%.0fm, normalized ANND=%.4f\n\n",
		result.BtoA.MeanDistMeters, result.BtoA.MedianDistMeters, result.BtoA.P90DistMeters, result.BtoA.NormalizedANND)
	fmt.Printf("Relationship: %s\n", result.Relationship)
	return nil
}
