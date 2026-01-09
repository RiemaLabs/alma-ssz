module alma.local/ssz

go 1.25.1

replace (
	alma.local/ssz => .
	github.com/ferranbt/fastssz => ./workspace/fastssz_bench
	github.com/ferranbt/fastssz v1.0.0 => ./workspace/fastssz_bench
)

replace github.com/prysmaticlabs/prysm/v5 => ./workspace/prysm

require (
	github.com/dave/dst v0.27.3
	github.com/ferranbt/fastssz v1.0.0
	github.com/prysmaticlabs/prysm/v5 v5.0.0-00010101000000-000000000000
)

require (
	github.com/OffchainLabs/go-bitfield v0.0.0-20251031151322-f427d04d8506 // indirect
	github.com/OffchainLabs/prysm/v7 v7.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.17.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/consensys/bavard v0.1.22 // indirect
	github.com/consensys/gnark-crypto v0.14.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/crate-crypto/go-kzg-4844 v1.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/emicklei/dot v1.9.1 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.15.9 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prysmaticlabs/fastssz v0.0.0-20251103153600-259302269bfc // indirect
	github.com/prysmaticlabs/gohashtree v0.0.5-beta // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/supranational/blst v0.3.16-0.20250831170142-f48500c1fdbe // indirect
	github.com/thomaso-mirodin/intmath v0.0.0-20160323211736-5dc6d854e46e // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
