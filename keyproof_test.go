package dotenv

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// Published PBKDF2-HMAC-SHA256 known-answer vectors (P="password", S="salt").
// These pin the cross-language contract anchor: PHP hash_pbkdf2 and JS WebCrypto
// deriveBits implement the same standard, so all three must agree on these.
func TestPBKDF2SHA256KnownAnswers(t *testing.T) {
	cases := []struct {
		iters  int
		dkLen  int
		expect string
	}{
		{1, 32, "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"},
		{2, 32, "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43"},
		{4096, 32, "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a"},
	}
	for _, tc := range cases {
		dk, err := pbkdf2.Key(sha256.New, "password", []byte("salt"), tc.iters, tc.dkLen)
		if err != nil {
			t.Fatalf("pbkdf2(iters=%d): %v", tc.iters, err)
		}
		got := hex.EncodeToString(dk)
		if got != tc.expect {
			t.Fatalf("pbkdf2(iters=%d): got %s, want %s", tc.iters, got, tc.expect)
		}
	}
}

// DeriveKeyProof must be deterministic and round-trip with GenerateKeyProof.
func TestKeyProofRoundTrip(t *testing.T) {
	const key = "52c7e8f043e61267076c35827d6c4be454c70ecac00bf10e79a56d703e32e123"

	salt, proof, iters, err := GenerateKeyProof(key)
	if err != nil {
		t.Fatalf("GenerateKeyProof: %v", err)
	}
	if iters != KeyProofIterations {
		t.Fatalf("iters = %d, want %d", iters, KeyProofIterations)
	}

	got, err := DeriveKeyProof(key, salt, iters)
	if err != nil {
		t.Fatalf("DeriveKeyProof: %v", err)
	}
	if got != proof {
		t.Fatalf("round-trip proof mismatch: got %s, want %s", got, proof)
	}

	// A wrong key must yield a different proof (this is the whole point).
	wrong, err := DeriveKeyProof("blah", salt, iters)
	if err != nil {
		t.Fatalf("DeriveKeyProof(wrong): %v", err)
	}
	if wrong == proof {
		t.Fatalf("wrong key produced the same proof — verification would be useless")
	}
}

// Cross-language contract anchor. This exact base64 proof is also asserted by
// the web app (apps/web EncryptionServiceTest) and the JS/PHP SDKs, locking
// byte-identical PBKDF2 output across every consumer.
//
//	key="test-key-123", salt=base64(16 zero bytes), iterations=1000
func TestKeyProofCrossLanguageVector(t *testing.T) {
	got, err := DeriveKeyProof("test-key-123", "AAAAAAAAAAAAAAAAAAAAAA==", 1000)
	if err != nil {
		t.Fatalf("DeriveKeyProof: %v", err)
	}
	const want = "NmvqZy0aWO8MAZ/l3xHShSRA3IRhdRwM6jCBBHDP+eE="
	if got != want {
		t.Fatalf("cross-language proof mismatch:\n got  %s\n want %s", got, want)
	}
}

// The proof is computed over padKey(key), so a 64-char key and its first-32-char
// truncation (which padKey maps to the same AES key) must produce the SAME proof.
func TestKeyProofUsesPaddedKey(t *testing.T) {
	const salt = "AAAAAAAAAAAAAAAAAAAAAA==" // 16 zero bytes, base64
	const iters = 1000

	full, err := DeriveKeyProof("52c7e8f043e61267076c35827d6c4be454c70ecac00bf10e79a56d703e32e123", salt, iters)
	if err != nil {
		t.Fatalf("DeriveKeyProof(full): %v", err)
	}
	truncated, err := DeriveKeyProof("52c7e8f043e61267076c35827d6c4be4", salt, iters)
	if err != nil {
		t.Fatalf("DeriveKeyProof(truncated): %v", err)
	}
	if full != truncated {
		t.Fatalf("proof differs for padKey-equivalent keys: %s vs %s", full, truncated)
	}
}
