// Package cmds provides CLI tools for running various actions
package cmds

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
)

// parseErrorThresholds parses a slice of "filename:percent" strings into a map.
// Percentages are specified as 0-100 (e.g., "stops.txt:5" means 5%).
// Use "*" as the filename for a default threshold.
// Example inputs: ["*:10", "stops.txt:5", "stop_times.txt:15"]
func parseErrorThresholds(thresholds []string) (map[string]float64, error) {
	if len(thresholds) == 0 {
		return nil, nil
	}
	result := make(map[string]float64)
	for _, t := range thresholds {
		parts := strings.SplitN(t, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid error threshold format '%s': expected 'filename:percent'", t)
		}
		filename := strings.TrimSpace(parts[0])
		percentStr := strings.TrimSpace(parts[1])
		if filename == "" {
			return nil, fmt.Errorf("invalid error threshold '%s': filename cannot be empty", t)
		}
		if percentStr == "" {
			return nil, fmt.Errorf("invalid error threshold '%s': percentage cannot be empty", t)
		}
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid error threshold percentage '%s': %w", percentStr, err)
		}
		if percent < 0 {
			return nil, fmt.Errorf("error threshold percentage cannot be negative: %s", t)
		}
		result[filename] = percent
	}
	return result, nil
}

// parseSecretEnv parses "target:ENV_VAR"; target is a feed_id, or a filename
// if it ends in .json. The key is read from the named env var.
func parseSecretEnv(arg string) (dmfr.Secret, error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return dmfr.Secret{}, fmt.Errorf("invalid --secret-env format %q: expected target:ENV_VAR", arg)
	}
	target := parts[0]
	envVar := parts[1]
	if target == "" || envVar == "" {
		return dmfr.Secret{}, fmt.Errorf("invalid --secret-env format %q: target and ENV_VAR must not be empty", arg)
	}
	key := os.Getenv(envVar)
	if key == "" {
		return dmfr.Secret{}, fmt.Errorf("environment variable %q is not set or empty", envVar)
	}
	secret := dmfr.Secret{Key: key}
	if strings.HasSuffix(target, ".json") {
		secret.Filename = target
	} else {
		secret.FeedID = target
	}
	return secret, nil
}
