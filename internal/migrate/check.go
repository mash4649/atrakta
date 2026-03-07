package migrate

import (
	"fmt"

	"atrakta/internal/events"
)

func Check(repoRoot string) error {
	ev, err := events.ReadAll(repoRoot)
	if err != nil {
		return err
	}
	for i, e := range ev {
		sv, ok := e.Raw["schema_version"]
		if !ok {
			return fmt.Errorf("event[%d] missing schema_version", i)
		}
		n, ok := sv.(float64)
		if !ok {
			return fmt.Errorf("event[%d] schema_version invalid", i)
		}
		if int(n) != events.SchemaVersion {
			return fmt.Errorf("event[%d] schema_version unsupported: got=%d want=%d", i, int(n), events.SchemaVersion)
		}
	}
	return nil
}
