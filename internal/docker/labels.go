package docker

const (
	LabelManaged           = "com.spin.managed"
	LabelName              = "com.spin.name"
	LabelService           = "com.spin.service"
	LabelPort            = "com.spin.port"
	LabelKafkaWithUI     = "com.spin.kafka.with-ui"
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
