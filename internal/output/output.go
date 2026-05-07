package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	Raw   Format = "raw"
)

func WriteJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func SaveJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteJSON(f, data)
}

func TableRows(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	writeRow(w, headers, widths)
	sep := make([]string, len(widths))
	for i, width := range widths {
		sep[i] = strings.Repeat("-", width)
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		writeRow(w, row, widths)
	}
}

func KeyValues(w io.Writer, rows [][2]string) {
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	for _, row := range rows {
		fmt.Fprintf(w, "%-*s  %s\n", width, row[0]+":", row[1])
	}
}

func FormatDurationUS(us int64) string {
	if us < 0 {
		return "-"
	}
	d := time.Duration(us) * time.Microsecond
	switch {
	case us < 1000:
		return fmt.Sprintf("%dus", us)
	case us < 1_000_000:
		return fmt.Sprintf("%.1fms", float64(us)/1000)
	case us < 60_000_000:
		return fmt.Sprintf("%.1fs", float64(us)/1_000_000)
	default:
		min := d / time.Minute
		sec := (d % time.Minute) / time.Second
		return fmt.Sprintf("%dm%02ds", min, sec)
	}
}

func RelativeTime(value string, now time.Time) string {
	if value == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	diff := now.Sub(t)
	if diff < 0 {
		diff = -diff
		return "in " + roundRelative(diff)
	}
	return roundRelative(diff) + " ago"
}

func Truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func StringMapLines(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("    %-24s = %v", key, values[key]))
	}
	return lines
}

func BoolYes(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func Percent(numerator, denominator float64) string {
	if denominator == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", numerator/denominator*100)
}

func ParseDurationToUS(value string) (int64, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	return d.Microseconds(), nil
}

func ToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func writeRow(w io.Writer, row []string, widths []int) {
	for i, width := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprintf(w, "%-*s", width, cell)
	}
	fmt.Fprintln(w)
}

func roundRelative(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
