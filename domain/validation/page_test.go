package validation

import "testing"

func TestValidatePageInputBoundaries(t *testing.T) {
	for _, slug := range []string{"notes/first-page", "decisions/core-v2", "a1"} {
		if err := ValidatePageSlug(slug); err != nil {
			t.Errorf("valid slug %q: %v", slug, err)
		}
	}
	for _, slug := range []string{"", "Notes/core", "notes/a b", "/notes/a", "notes//a", "notes/a/"} {
		if err := ValidatePageSlug(slug); err == nil {
			t.Errorf("invalid slug accepted: %q", slug)
		}
	}
	if err := ValidateTimelineEntry("2026-08-17\nchanged", false); err == nil {
		t.Fatal("multiline timeline entry accepted")
	}
	if err := ValidateCompiledTruth("body\n<!-- timeline -->\nhidden"); err == nil {
		t.Fatal("timeline delimiter accepted in compiled truth")
	}
}
