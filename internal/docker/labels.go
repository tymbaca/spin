package docker

const (
	LabelManaged                    = "com.spin.managed"
	LabelName                       = "com.spin.name"
	LabelService                    = "com.spin.service"
	LabelPort                       = "com.spin.port"
	LabelCredentialPostgresUser     = "com.spin.credentials.postgres.user"
	LabelCredentialPostgresPassword = "com.spin.credentials.postgres.password"
	LabelCredentialPostgresDatabase = "com.spin.credentials.postgres.database"
	LabelCredentialKafkaUser        = "com.spin.credentials.kafka.user"
	LabelCredentialKafkaPassword    = "com.spin.credentials.kafka.password"
	LabelCredentialKafkaProtocol    = "com.spin.credentials.kafka.protocol"
	LabelCredentialKafkaMechanism   = "com.spin.credentials.kafka.mechanism"
	LabelKafkaWithUI                = "com.spin.kafka.with-ui"
)

const (
	ServicePostgres = "postgres"
	ServiceKafka    = "kafka"
	ServiceKafkaUI  = "kafka-ui"
)

func KafkaUIName(kafkaName string) string {
	return kafkaName + "-ui"
}

func ContainerName(name string) string {
	return "spin-" + name
}

func VolumeName(name string) string {
	return "spin-" + name + "-data"
}
