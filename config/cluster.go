package config

const (
	ClientPort = "127.0.0.1:8848"

	ClusterPort = "127.0.0.1:8849"
)

func GetAddrList() []string {
	return []string{"127.0.0.1:8849", "127.0.0.1:8849"}
}

func GetLeaderIp() string {
	return "127.0.0.1"
}

func GetThisIp() string {
	return "127.0.0.1"
}
