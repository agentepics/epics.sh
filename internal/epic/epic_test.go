package epic

import "testing"

func TestParseFrontmatterReturnsPartialMapWhenUnclosed(t *testing.T) {
	values := parseFrontmatter("---\nid: partial-epic\nsummary: still-present\n")
	if values["id"] != "partial-epic" {
		t.Fatalf("expected partial id, got %#v", values)
	}
	if values["summary"] != "still-present" {
		t.Fatalf("expected partial summary, got %#v", values)
	}
}
