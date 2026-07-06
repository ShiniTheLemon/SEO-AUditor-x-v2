package identity

import "testing"

func TestUserIDFromFingerprint(t *testing.T) {
	first := UserIDFromFingerprint(" test-device ")
	second := UserIDFromFingerprint("test-device")

	if first == "" {
		t.Fatal("expected user ID")
	}

	if first != second {
		t.Fatal("expected stable normalized hash")
	}

	if first == "test-device" {
		t.Fatal("expected hashed user ID")
	}
}

func TestUserIDFromFingerprintRequiresValue(t *testing.T) {
	if UserIDFromFingerprint(" ") != "" {
		t.Fatal("expected empty user ID for blank fingerprint")
	}
}
