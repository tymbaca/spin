package docker

const (
	LabelManaged = "com.spin.managed"
	LabelName    = "com.spin.name"
	LabelService = "com.spin.service"
	LabelPort    = "com.spin.port"
)

const (
	ServicePostgres = "postgres"
)

func ContainerName(name string) string {
	return "spin-" + name
}

func VolumeName(name string) string {
	return "spin-" + name + "-data"
}
