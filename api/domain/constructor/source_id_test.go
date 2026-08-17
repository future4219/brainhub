package constructor

import "testing"

func TestNewSourceID(t *testing.T) {
	for _, value := range []string{"a", "brainhub", "a-123", "12345678901234567890123456789012"} {
		if _, err := NewSourceID(value); err != nil {
			t.Errorf("NewSourceID(%q): %v", value, err)
		}
	}
	for _, value := range []string{"", "default", "BrainHub", "brain_hub", "brain/hub", "123456789012345678901234567890123"} {
		if _, err := NewSourceID(value); err == nil {
			t.Errorf("NewSourceID(%q) succeeded; want error", value)
		}
	}
}
