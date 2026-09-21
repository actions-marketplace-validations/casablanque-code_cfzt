// Package validate holds shared input-validation rules used across cfzt's
// command layer. TunnelName in particular is the single source of truth for
// what counts as a valid tunnel/service name — the same string is used as a
// filesystem directory name, a DNS hostname label, a systemd unit name, and a
// state map key, so it must be safe in all of those contexts at once.
package validate

import (
	"fmt"
	"regexp"
)

// nameRE mirrors DNS label rules (which are the tightest constraint among
// filesystem/systemd/DNS): lowercase alphanumeric, hyphens allowed in the
// middle, must start with an alphanumeric character, max 63 chars.
var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// TunnelName validates a tunnel/service name. It is used as a filesystem
// path component, a DNS hostname prefix, and a systemd unit name, so it must
// reject anything that could act as a path traversal sequence, a shell/unit
// metacharacter, or an invalid DNS label.
func TunnelName(name string) error {
	if name == "" {
		return fmt.Errorf("tunnel name cannot be empty")
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf(
			"invalid tunnel name %q: must match %s (lowercase letters, digits, hyphens; must start with a letter or digit; max 63 chars)",
			name, nameRE.String(),
		)
	}
	return nil
}
