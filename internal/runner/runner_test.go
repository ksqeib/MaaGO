package runner

import "testing"

func TestParseLogLine(t *testing.T) {
	entry, ok := parseLogLine("16:07:54 [info] 开始：理智作战")
	if !ok {
		t.Fatal("expected a parsed log line")
	}
	if entry.Time != "16:07:54" || entry.Level != "info" || entry.Message != "开始：理智作战" {
		t.Fatalf("unexpected entry: %+v", entry)
	}

	if _, ok := parseLogLine("not a log line"); ok {
		t.Fatal("malformed line should be rejected")
	}
}
