package helpers

import "time"

func FormatBirthday(t *time.Time) *string {
	if t == nil {
		return nil
	}

	formatted := t.Format("2006-01-02")
	return &formatted
}
