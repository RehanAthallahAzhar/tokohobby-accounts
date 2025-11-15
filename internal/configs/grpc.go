package configs

type GrpcConfig struct {
	AccountServiceAddress string `env:"ACCOUNT_GRPC_SERVER_ADDRESS,required"`
	ProductServiceAddress string `env:"PRODUCT_SERVICE_GRPC_URL,required"`
}
