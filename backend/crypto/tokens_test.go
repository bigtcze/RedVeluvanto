package crypto

import (
	"encoding/base64"
	"os"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
	}{
		{"empty passphrase", ""},
		{"short passphrase", "key"},
		{"normal passphrase", "my-secret-key-123"},
		{"long passphrase", "this-is-a-very-long-passphrase-that-exceeds-32-bytes-significantly"},
		{"unicode passphrase", "klíč-šifrování-🔑"},
		{"special chars", "p@$$w0rd!#%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveKey(tt.passphrase)

			if len(result) != 32 {
				t.Errorf("DeriveKey(%q) returned %d bytes, want 32", tt.passphrase, len(result))
			}
		})
	}

	t.Run("deterministic", func(t *testing.T) {
		a := DeriveKey("test-key")
		b := DeriveKey("test-key")
		if string(a) != string(b) {
			t.Error("DeriveKey is not deterministic: same input produced different outputs")
		}
	})

	t.Run("different inputs produce different outputs", func(t *testing.T) {
		a := DeriveKey("key-one")
		b := DeriveKey("key-two")
		if string(a) == string(b) {
			t.Error("DeriveKey produced identical output for different inputs")
		}
	})
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		key       string
	}{
		{"empty plaintext", "", "secret"},
		{"short text", "hi", "secret"},
		{"normal text", "Hello, World!", "my-key"},
		{"long text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.", "key123"},
		{"unicode text", "Příliš žluťoučký kůň úpěl ďábelské ódy 🇨🇿", "unicode-key"},
		{"special characters", `!@#$%^&*()_+-={}[]|\:;"'<>,.?/~` + "`", "spec-key"},
		{"newlines and tabs", "line1\nline2\ttab", "nl-key"},
		{"json payload", `{"token":"abc","refresh":"xyz"}`, "json-key"},
		{"empty key", "some data", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, tt.key)
			if err != nil {
				t.Fatalf("Encrypt(%q, %q) error: %v", tt.plaintext, tt.key, err)
			}

			if encrypted == tt.plaintext && tt.plaintext != "" {
				t.Error("Encrypt returned plaintext unchanged")
			}

			_, err = base64.StdEncoding.DecodeString(encrypted)
			if err != nil {
				t.Errorf("Encrypt output is not valid base64: %v", err)
			}

			decrypted, err := Decrypt(encrypted, tt.key)
			if err != nil {
				t.Fatalf("Decrypt error: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("roundtrip failed: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	ct1, err := Encrypt("same", "same-key")
	if err != nil {
		t.Fatal(err)
	}
	ct2, err := Encrypt("same", "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if ct1 == ct2 {
		t.Error("Encrypt produced identical ciphertext for same input (nonce should differ)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt("secret data", "correct-key")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(encrypted, "wrong-key")
	if err == nil {
		t.Error("Decrypt with wrong key should return error")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not base64 at all", "this is not base64!!!"},
		{"invalid padding", "aGVsbG8"},
		{"invalid characters", "====????"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, "any-key")
			if err == nil {
				t.Errorf("Decrypt(%q) should return error for invalid base64", tt.ciphertext)
			}
		})
	}
}

func TestDecryptTooShortCiphertext(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("tiny"))
	_, err := Decrypt(short, "any-key")
	if err == nil {
		t.Error("Decrypt should return error for ciphertext shorter than nonce size")
	}
}

func TestGetEncKey(t *testing.T) {
	tests := []struct {
		name          string
		tokenKey      string
		litellmKey    string
		expectedValue string
	}{
		{"TOKEN_ENCRYPTION_KEY set", "my-token-key", "my-litellm-key", "my-token-key"},
		{"TOKEN_ENCRYPTION_KEY empty, LITELLM_MASTER_KEY set", "", "my-litellm-key", "my-litellm-key"},
		{"both empty falls back to default", "", "", "redveluvanto-default-key"},
		{"TOKEN_ENCRYPTION_KEY takes precedence", "token-key", "litellm-key", "token-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origToken := os.Getenv("TOKEN_ENCRYPTION_KEY")
			origLitellm := os.Getenv("LITELLM_MASTER_KEY")
			t.Cleanup(func() {
				os.Setenv("TOKEN_ENCRYPTION_KEY", origToken)
				os.Setenv("LITELLM_MASTER_KEY", origLitellm)
			})

			os.Setenv("TOKEN_ENCRYPTION_KEY", tt.tokenKey)
			os.Setenv("LITELLM_MASTER_KEY", tt.litellmKey)

			got := GetEncKey()
			if got != tt.expectedValue {
				t.Errorf("GetEncKey() = %q, want %q", got, tt.expectedValue)
			}
		})
	}
}
