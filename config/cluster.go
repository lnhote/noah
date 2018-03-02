package config

const (
	ClientPort = "127.0.0.1:8848"

	ClusterPort = "127.0.0.1:8849"
)

func GetServerList() []string {
	return []string{"127.0.0.1:8849", "127.0.0.1:8849"}
}

func GetLeaderAddr() string {
	return "127.0.0.1:8849"
}

func GetLocalAddr() string {
	return "127.0.0.1:8849"
}
