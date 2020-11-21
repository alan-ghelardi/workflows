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
		in   string
		want string
	}{
		{in: "text", want: "text"},
		{in: "{.head_commit.id}", want: "833568e"},
		{in: "{.head_commit}", want: `{"id":"833568e"}`},
		{in: "{.commits..modified}", want: `[["README.md"],["CHANGELOG.md"]]`},
		{in: "{.forced}", want: "false"},
		{in: "{.repository.pushed_at}", want: "1613517959"},
		{in: "{.base_ref}", want: null},
		{in: "{.missing_key}", want: "[]"},
		{in: "{.commits..missing_key}", want: "[]"},
	}

	for _, test := range tests {
		gotResult, err := query(event, test.in)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(test.want, gotResult); diff != "" {
			t.Errorf("Mismatch (-want +got):\n%s", diff)
		}
	}
}
