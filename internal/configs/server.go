package configs

type ServerConfig struct {
	Port        string `env:"SERVER_PORT,required"`
	GRPCPort    string `env:"GRPC_PORT,required"`
	JWTSecret   string `env:"JWT_SECRET,required"`
	JWTIssuer   string `env:"JWT_ISSUER,required"`
	JWTAudience string `env:"JWT_AUDIENCE,required"`
}
