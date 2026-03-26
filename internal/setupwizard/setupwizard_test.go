package setupwizard

import (
	"bufio"
	"strings"
	"testing"
)

func reader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

// --- askString ---

func TestAskString_EmptyInputReturnsDefault(t *testing.T) {
	result := askString(reader("\n"), "label", "default-value")
	if result != "default-value" {
		t.Errorf("askString() empty input = %q, want %q", result, "default-value")
	}
}

func TestAskString_NewValueReturnsInput(t *testing.T) {
	result := askString(reader("new-value\n"), "label", "old")
	if result != "new-value" {
		t.Errorf("askString() = %q, want %q", result, "new-value")
	}
}

func TestAskString_TrimsWhitespace(t *testing.T) {
	result := askString(reader("  trimmed  \n"), "label", "old")
	if result != "trimmed" {
		t.Errorf("askString() = %q, want %q", result, "trimmed")
	}
}

// --- askInt ---

func TestAskInt_EmptyInputReturnsDefault(t *testing.T) {
	result := askInt(reader("\n"), "label", 42)
	if result != 42 {
		t.Errorf("askInt() empty input = %d, want 42", result)
	}
}

func TestAskInt_ValidIntReturnsValue(t *testing.T) {
	result := askInt(reader("7\n"), "label", 42)
	if result != 7 {
		t.Errorf("askInt() = %d, want 7", result)
	}
}

func TestAskInt_InvalidThenValidReturnsValid(t *testing.T) {
	// "abc" 실패 후 "5" 입력
	result := askInt(reader("abc\n5\n"), "label", 0)
	if result != 5 {
		t.Errorf("askInt() after invalid = %d, want 5", result)
	}
}

func TestAskInt_ZeroIsValid(t *testing.T) {
	result := askInt(reader("0\n"), "label", 9)
	if result != 0 {
		t.Errorf("askInt() zero = %d, want 0", result)
	}
}

// --- askYesNo ---

func TestAskYesNo_EmptyInputReturnsDefault(t *testing.T) {
	result := askYesNo(reader("\n"), "label", true)
	if !result {
		t.Errorf("askYesNo() empty with default true = false, want true")
	}

	result2 := askYesNo(reader("\n"), "label", false)
	if result2 {
		t.Errorf("askYesNo() empty with default false = true, want false")
	}
}

func TestAskYesNo_YReturnsTrue(t *testing.T) {
	for _, input := range []string{"y\n", "yes\n", "Y\n", "YES\n"} {
		result := askYesNo(reader(input), "label", false)
		if !result {
			t.Errorf("askYesNo(%q) = false, want true", input)
		}
	}
}

func TestAskYesNo_NReturnsFalse(t *testing.T) {
	for _, input := range []string{"n\n", "no\n", "N\n", "NO\n"} {
		result := askYesNo(reader(input), "label", true)
		if result {
			t.Errorf("askYesNo(%q) = true, want false", input)
		}
	}
}

func TestAskYesNo_InvalidThenYReturnsTrue(t *testing.T) {
	result := askYesNo(reader("maybe\ny\n"), "label", false)
	if !result {
		t.Errorf("askYesNo() after invalid = false, want true")
	}
}
