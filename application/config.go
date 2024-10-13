package application

type Config struct {
	RedisAddress string
	ServerPort   uint64
}

func LoadConfig() Config {
	cfg := Config{
		RedisAddress: "localhost:6379",
		ServerPort:   3000,
	}

	// if redisAddr, exists := os.LookupEnv("REDDIS_ADDR"); exists {
	// 	cfg.RedisAddress = redisAddr
	// }

	// if serverPort, exists := os.LookupEnv("SERVER_PORT"); exists {
	// 	if port, err := strconv.ParseUint(serverPort, 10, 16); err == nil {
	// 		cfg.ServerPort = uint64(port)
	// 	}
	// }

	return cfg
}
