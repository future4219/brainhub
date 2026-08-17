package validation

import (
	"strings"
	"testing"
)

func TestPasswordLengthUsesBytes(t *testing.T) {
	for _, password := range []string{"123456789012", strings.Repeat("a", 72), "あいうえ"} {
		if err := ValidatePassword(password); err != nil {
			t.Errorf("ValidatePassword length %d: %v", len(password), err)
		}
	}
	for _, password := range []string{"12345678901", strings.Repeat("a", 73), "あいう"} {
		if err := ValidatePassword(password); err == nil {
			t.Errorf("ValidatePassword length %d succeeded", len(password))
		}
	}
}
