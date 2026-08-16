package constructor

import (
	"errors"
	"strings"
	"testing"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/domain/validation"
)

func TestNewBrainCreate(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	brain, err := NewBrainCreate("brain-id", entity.SourceID("accounting"), "Accounting", "notes", "", "user-id", now)
	if err != nil {
		t.Fatal(err)
	}
	if brain.State != entconst.BrainStateProvisioning || brain.Visibility != entconst.VisibilityPrivate {
		t.Fatalf("unexpected defaults: state=%q visibility=%q", brain.State, brain.Visibility)
	}
	membership := NewOwnerMembership("membership-id", brain.ID, brain.OwnerID, now)
	if membership.Role != entity.RoleOwner || !membership.IsActive() {
		t.Fatalf("unexpected owner membership: %#v", membership)
	}
	adopted, err := NewBrainAdopt("adopted-id", entity.SourceID("existing"), "Existing", "", "", "user-id", now)
	if err != nil {
		t.Fatal(err)
	}
	if adopted.State != entconst.BrainStateReady || adopted.Visibility != entconst.VisibilityPrivate {
		t.Fatalf("unexpected adopt defaults: state=%q visibility=%q", adopted.State, adopted.Visibility)
	}
}

func TestNewBrainCreateValidation(t *testing.T) {
	tests := []struct {
		name        string
		brainName   string
		description string
		visibility  entconst.Visibility
		want        error
	}{
		{name: "empty name", description: "ok", want: validation.ErrInvalidBrainName},
		{name: "long name", brainName: strings.Repeat("a", 81), want: validation.ErrInvalidBrainName},
		{name: "control", brainName: "bad\nname", want: validation.ErrInvalidBrainName},
		{name: "long description", brainName: "ok", description: strings.Repeat("a", 501), want: validation.ErrInvalidBrainDescription},
		{name: "invalid visibility", brainName: "ok", visibility: "unlisted", want: ErrInvalidVisibility},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewBrainCreate("id", entity.SourceID("source"), test.brainName, test.description, test.visibility, "user", time.Time{})
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
