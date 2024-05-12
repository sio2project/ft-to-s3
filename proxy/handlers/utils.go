package handlers

import (
	"net/http"
	"time"
)

func set_content_type_json(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func fromRFC2822(date string) (int64, error) {
	const rfc2822 = "Mon, 02 Jan 2006 15:04:05 MST"
	t, err := time.Parse(rfc2822, date)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func toRFC2822(unix int64) string {
	return time.Unix(unix, 0).Format(time.RFC1123)
}
