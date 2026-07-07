package cli

import (
	"testing"

	"github.com/tymbaca/spin/internal/docker"
)

func TestCredentialsForRedis(t *testing.T) {
	got := credentialsFor(docker.ContainerInfo{
		Service: docker.ServiceRedis,
		Port:    6379,
	})

	const want = "redis://127.0.0.1:6379"
	if got != want {
		t.Fatalf("credentialsFor(redis) = %q, want %q", got, want)
	}
}

func TestCredentialsForRedisWithPassword(t *testing.T) {
	got := credentialsFor(docker.ContainerInfo{
		Service: docker.ServiceRedis,
		Port:    6379,
		Credentials: docker.CredentialInfo{
			Password: "secret",
		},
	})

	const want = "redis://:secret@127.0.0.1:6379"
	if got != want {
		t.Fatalf("credentialsFor(redis with password) = %q, want %q", got, want)
	}
}

func TestCredentialsForRedisEscapesPassword(t *testing.T) {
	got := credentialsFor(docker.ContainerInfo{
		Service: docker.ServiceRedis,
		Port:    6379,
		Credentials: docker.CredentialInfo{
			Password: "p@ss word",
		},
	})

	const want = "redis://:p%40ss%20word@127.0.0.1:6379"
	if got != want {
		t.Fatalf("credentialsFor(redis with escaped password) = %q, want %q", got, want)
	}
}
