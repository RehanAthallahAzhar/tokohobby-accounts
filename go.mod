module github.com/RehanAthallahAzhar/tokohobby-accounts

go 1.24.5

require (
	github.com/RehanAthallahAzhar/tokohobby-messaging v0.5.3
	github.com/RehanAthallahAzhar/tokohobby-protos v0.0.1
	github.com/caarlos0/env/v6 v6.10.1
	github.com/go-playground/validator/v10 v10.30.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/labstack/echo/v4 v4.15.0
	github.com/labstack/gommon v0.4.2
	github.com/lib/pq v1.10.9
	github.com/sirupsen/logrus v1.9.4
	golang.org/x/crypto v0.47.0
	google.golang.org/grpc v1.78.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/segmentio/kafka-go v0.4.50 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120174246-409b4a993575 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/RehanAthallahAzhar/tokohobby-messaging => ../messaging
	github.com/RehanAthallahAzhar/tokohobby-protos => ../protos
)
