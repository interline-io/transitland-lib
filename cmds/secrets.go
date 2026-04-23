package cmds

import (
	"fmt"
	"os"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
)

// parseSecretEnv parses a --secret-env argument in the format "target:ENV_VAR"
// where target is either a feed_id or a filename (detected by .json suffix).
// The secret key value is read from the environment variable.
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
