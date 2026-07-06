package kafka

import "testing"

func TestCredentialsWithAuthUsesCustomSASLCredentials(t *testing.T) {
	got := CredentialsWithAuth(AuthConfig{
		Port:      9093,
		Protocol:  "SASL_PLAINTEXT",
		Mechanism: "PLAIN",
		User:      "myuser",
		Password:  "mypass",
	})
	want := "127.0.0.1:9093 SASL_PLAINTEXT PLAIN username=myuser password=mypass"

	if got != want {
		t.Fatalf("CredentialsWithAuth() = %q, want %q", got, want)
	}
}

func TestCredentialsWithAuthPlaintextOmitsSASLCredentials(t *testing.T) {
	got := CredentialsWithAuth(AuthConfig{
		Port:     9092,
		Protocol: "PLAINTEXT",
	})
	want := "127.0.0.1:9092 PLAINTEXT"

	if got != want {
		t.Fatalf("CredentialsWithAuth() = %q, want %q", got, want)
	}
}
