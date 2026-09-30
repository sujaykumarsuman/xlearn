module github.com/sujaykumarsuman/xlearn

go 1.26.0

// npm packages occasionally ship Go files; keep ./... out of them.
ignore ./web/node_modules

require (
	github.com/criyle/go-sandbox v0.13.7
	github.com/jackc/pgx/v5 v5.11.0
	github.com/nats-io/nats-server/v2 v2.14.7
	github.com/nats-io/nats.go v1.54.0
	github.com/nats-io/nkeys v0.4.16
	github.com/pressly/goose/v3 v3.28.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/yuin/goldmark v1.8.6
	golang.org/x/crypto v0.57.0
	golang.org/x/sys v0.48.0
	golang.org/x/time v0.16.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/antithesishq/antithesis-sdk-go v0.8.0-default-no-op // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/nats-io/jwt/v2 v2.8.2 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/sethvargo/go-retry v0.4.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
