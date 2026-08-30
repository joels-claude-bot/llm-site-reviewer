package opencodereview

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sst/opencode-sdk-go"
)

func TestTraceRecorderUsesTypedToolStatesAndTimings(t *testing.T) {
	base := time.Unix(100, 0)
	times := []time.Time{base, base.Add(2 * time.Second), base.Add(2500 * time.Millisecond), base.Add(3 * time.Second)}
	n := 0
	now := func() time.Time { v := times[n]; n++; return v }
	var output bytes.Buffer
	recorder := newTraceRecorder(&output, now)

	recorder.record(decodeToolPart(t, `{"id":"call-1","callID":"c1","messageID":"m1","sessionID":"s1","tool":"grep","type":"tool","state":{"status":"running","input":{"pattern":"NANC"},"time":{"start":1000}}}`))
	// Repeated stream updates must not emit a second start line.
	recorder.record(decodeToolPart(t, `{"id":"call-1","callID":"c1","messageID":"m1","sessionID":"s1","tool":"grep","type":"tool","state":{"status":"running","input":{"pattern":"NANC"},"time":{"start":1000}}}`))
	recorder.record(decodeToolPart(t, `{"id":"call-1","callID":"c1","messageID":"m1","sessionID":"s1","tool":"grep","type":"tool","state":{"status":"completed","input":{"pattern":"NANC"},"output":"match","title":"grep","metadata":{},"time":{"start":1000,"end":1125}}}`))

	got := output.String()
	for _, want := range []string{"+2s (gap 2s) grep started", "+3s (gap 1s) grep completed in 125ms", `input={"pattern":"NANC"}`} {
		if !strings.Contains(got, want) {
			t.Fatalf("trace output %q does not contain %q", got, want)
		}
	}
	if strings.Count(got, "started") != 1 {
		t.Fatalf("duplicate start was logged: %q", got)
	}
}

func TestTraceRecorderIgnoresNonReadOnlyTools(t *testing.T) {
	base := time.Unix(100, 0)
	now := func() time.Time { return base }
	var output bytes.Buffer
	recorder := newTraceRecorder(&output, now)
	recorder.record(decodeToolPart(t, `{"id":"call-1","callID":"c1","messageID":"m1","sessionID":"s1","tool":"bash","type":"tool","state":{"status":"running","input":{},"time":{"start":1000}}}`))
	if output.Len() != 0 {
		t.Fatalf("unexpected trace: %s", output.String())
	}
}

func decodeToolPart(t *testing.T, value string) opencode.ToolPart {
	t.Helper()
	var part opencode.ToolPart
	if err := json.Unmarshal([]byte(value), &part); err != nil {
		t.Fatal(err)
	}
	return part
}
