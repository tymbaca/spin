package docker

import "testing"

func TestCredentialsFromContainerPrefersLabels(t *testing.T) {
	credentials := credentialsFromContainer(ServicePostgres, map[string]string{
		LabelCredentialPostgresUser:     "saved-user",
		LabelCredentialPostgresPassword: "saved-password",
		LabelCredentialPostgresDatabase: "saved-db",
	}, []string{
		"POSTGRES_USER=env-user",
		"POSTGRES_PASSWORD=env-password",
		"POSTGRES_DB=env-db",
	})

	if credentials.User != "saved-user" {
		t.Fatalf("User = %q, want %q", credentials.User, "saved-user")
	}
	if credentials.Password != "saved-password" {
		t.Fatalf("Password = %q, want %q", credentials.Password, "saved-password")
	}
	if credentials.Database != "saved-db" {
		t.Fatalf("Database = %q, want %q", credentials.Database, "saved-db")
	}
}

func TestCredentialsFromContainerFallsBackToPostgresEnv(t *testing.T) {
	credentials := credentialsFromContainer(ServicePostgres, nil, []string{
		"POSTGRES_USER=app",
		"POSTGRES_PASSWORD=secret",
		"POSTGRES_DB=appdb",
	})

	if credentials.User != "app" {
		t.Fatalf("User = %q, want %q", credentials.User, "app")
	}
	if credentials.Password != "secret" {
		t.Fatalf("Password = %q, want %q", credentials.Password, "secret")
	}
	if credentials.Database != "appdb" {
		t.Fatalf("Database = %q, want %q", credentials.Database, "appdb")
	}
}

func TestCredentialsFromContainerFallsBackToKafkaEnv(t *testing.T) {
	credentials := credentialsFromContainer(ServiceKafka, nil, []string{
		"KAFKA_ADVERTISED_LISTENERS=HOST://127.0.0.1:9092,DOCKER://spin-kafka:9094",
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=HOST:SASL_PLAINTEXT,DOCKER:SASL_PLAINTEXT,CONTROLLER:PLAINTEXT",
		"KAFKA_SASL_ENABLED_MECHANISMS=PLAIN",
		`KAFKA_LISTENER_NAME_HOST_PLAIN_SASL_JAAS_CONFIG=org.apache.kafka.common.security.plain.PlainLoginModule required username="myuser" password="mypass";`,
	})

	if credentials.Protocol != "SASL_PLAINTEXT" {
		t.Fatalf("Protocol = %q, want %q", credentials.Protocol, "SASL_PLAINTEXT")
	}
	if credentials.Mechanism != "PLAIN" {
		t.Fatalf("Mechanism = %q, want %q", credentials.Mechanism, "PLAIN")
	}
	if credentials.User != "myuser" {
		t.Fatalf("User = %q, want %q", credentials.User, "myuser")
	}
	if credentials.Password != "mypass" {
		t.Fatalf("Password = %q, want %q", credentials.Password, "mypass")
	}
}

func TestCredentialsFromContainerReadsRedisPasswordLabel(t *testing.T) {
	credentials := credentialsFromContainer(ServiceRedis, map[string]string{
		LabelCredentialRedisPassword: "secret",
	}, nil)

	if credentials.Password != "secret" {
		t.Fatalf("Password = %q, want %q", credentials.Password, "secret")
	}
}
