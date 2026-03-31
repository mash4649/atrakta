package audit

import "fmt"

// AppendAndVerify appends one audit event and immediately verifies integrity at the same level.
func AppendAndVerify(storeDir, level, action string, payload any) (Event, error) {
	ev, err := AppendEvent(storeDir, level, action, payload)
	if err != nil {
		return Event{}, err
	}
	if err := VerifyIntegrity(storeDir, level); err != nil {
		return Event{}, fmt.Errorf("audit verify after append: %w", err)
	}
	return ev, nil
}
