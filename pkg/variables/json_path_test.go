package variables

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nubank/workflows/pkg/testutils"
)

func TestQuery(t *testing.T) {
	event, err := testutils.ReadEvent("push-event.json")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		expr   string
		result string
	}{
		{"text", "text"},
		{"{.head_commit.id}", "833568e"},
		{"{.head_commit}", `{"id":"833568e"}`},
		{"{.commits..modified}", `[["README.md"],["CHANGELOG.md"]]`},
		{"{.forced}", "false"},
		{"{.base_ref}", null},
		{"{.missing_key}", "[]"},
		{"{.commits..missing_key}", "[]"},
	}

	for _, test := range tests {
		gotResult, err := query(event, test.expr)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(test.result, gotResult); diff != "" {
			t.Errorf("Mismatch (-want +got):\n%s", diff)
		}
	}
}
