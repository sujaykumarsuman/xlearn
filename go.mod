module github.com/sujaykumarsuman/xlearn

go 1.26.0

// npm packages occasionally ship Go files; keep ./... out of them.
ignore ./web/node_modules

require (
	github.com/jackc/pgx/v5 v5.11.0
	github.com/nats-io/nats-server/v2 v2.10.22
	github.com/nats-io/nats.go v1.54.0
	github.com/pressly/goose/v3 v3.28.0
	golang.org/x/crypto v0.57.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/nats-io/jwt/v2 v2.5.8 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/time v0.7.0 // indirect
)
