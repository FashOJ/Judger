module github.com/FashOJ/Judger

go 1.24.0

require (
	github.com/redis/go-redis/v9 v9.17.1
	github.com/seccomp/libseccomp-golang v0.11.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
)

replace github.com/FashOJ/Judger => ./

replace github.com/FashOJ/Judger/api/proto/judger => ./api/proto/judger
