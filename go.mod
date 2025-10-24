module github.com/opencloud-eu/opencloud

go 1.24.6

require (
	dario.cat/mergo v1.0.2
	github.com/CiscoM31/godata v1.0.11
	github.com/KimMachineGun/automemlimit v0.7.4
	github.com/Masterminds/semver v1.5.0
	github.com/MicahParks/keyfunc/v2 v2.1.0
	github.com/Nerzal/gocloak/v13 v13.9.0
	github.com/bbalet/stopwords v1.0.0
	github.com/beevik/etree v1.6.0
	github.com/blevesearch/bleve/v2 v2.5.4
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/coreos/go-oidc/v3 v3.16.0
	github.com/cs3org/go-cs3apis v0.0.0-20250908152307-4ca807afe54e
	github.com/davidbyttow/govips/v2 v2.16.0
	github.com/dhowden/tag v0.0.0-20240417053706-3d75831295e8
	github.com/dutchcoders/go-clamd v0.0.0-20170520113014-b970184f4d9e
	github.com/gabriel-vasile/mimetype v1.4.10
	github.com/ggwhite/go-masker v1.1.0
	github.com/go-chi/chi/v5 v5.2.3
	github.com/go-chi/render v1.0.3
	github.com/go-jose/go-jose/v3 v3.0.4
	github.com/go-ldap/ldap/v3 v3.4.12
	github.com/go-ldap/ldif v0.0.0-20200320164324-fd88d9b715b3
	github.com/go-micro/plugins/v4/client/grpc v1.2.1
	github.com/go-micro/plugins/v4/logger/zerolog v1.2.0
	github.com/go-micro/plugins/v4/registry/memory v1.2.0
	github.com/go-micro/plugins/v4/server/grpc v1.2.0
	github.com/go-micro/plugins/v4/server/http v1.2.2
	github.com/go-micro/plugins/v4/store/nats-js-kv v0.0.0-20240726082623-6831adfdcdc4
	github.com/go-micro/plugins/v4/wrapper/monitoring/prometheus v1.2.0
	github.com/go-micro/plugins/v4/wrapper/trace/opentelemetry v1.2.0
	github.com/go-playground/validator/v10 v10.28.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.7.0
	github.com/google/go-tika v0.3.1
	github.com/google/uuid v1.6.0
	github.com/gookit/config/v2 v2.2.7
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3
	github.com/invopop/validation v0.8.0
	github.com/jellydator/ttlcache/v2 v2.11.1
	github.com/jellydator/ttlcache/v3 v3.4.0
	github.com/jinzhu/now v1.1.5
	github.com/justinas/alice v1.2.0
	github.com/kovidgoyal/imaging v1.7.2
	github.com/leonelquinteros/gotext v1.7.2
	github.com/libregraph/idm v0.5.0
	github.com/libregraph/lico v0.66.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mna/pigeon v1.3.0
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/nats-io/nats-server/v2 v2.12.0
	github.com/nats-io/nats.go v1.46.1
	github.com/oklog/run v1.2.0
	github.com/olekukonko/tablewriter v1.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/ginkgo/v2 v2.26.0
	github.com/onsi/gomega v1.38.2
	github.com/open-policy-agent/opa v1.9.0
	github.com/opencloud-eu/icap-client v0.0.0-20250930132611-28a2afe62d89
	github.com/opencloud-eu/libre-graph-api-go v1.0.8-0.20250724122329-41ba6b191e76
	github.com/opencloud-eu/reva/v2 v2.39.1-0.20251023125727-bf6471473de6
	github.com/opensearch-project/opensearch-go/v4 v4.5.0
	github.com/orcaman/concurrent-map v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/xattr v0.4.12
	github.com/prometheus/client_golang v1.23.2
	github.com/r3labs/sse/v2 v2.10.0
	github.com/riandyrn/otelchi v0.12.2
	github.com/rogpeppe/go-internal v1.14.1
	github.com/rs/cors v1.11.1
	github.com/rs/zerolog v1.34.0
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af
	github.com/spf13/afero v1.15.0
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	github.com/test-go/testify v1.1.4
	github.com/testcontainers/testcontainers-go v0.39.0
	github.com/testcontainers/testcontainers-go/modules/opensearch v0.39.0
	github.com/theckman/yacspin v0.13.12
	github.com/thejerf/suture/v4 v4.0.6
	github.com/tidwall/gjson v1.18.0
	github.com/tidwall/sjson v1.2.5
	github.com/tus/tusd/v2 v2.8.0
	github.com/unrolled/secure v1.16.0
	github.com/urfave/cli/v2 v2.27.7
	github.com/vmihailenco/msgpack/v5 v5.4.1
	github.com/xhit/go-simple-mail/v2 v2.16.0
	go-micro.dev/v4 v4.11.0
	go.etcd.io/bbolt v1.4.3
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0
	go.opentelemetry.io/contrib/zpages v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	golang.org/x/crypto v0.43.0
	golang.org/x/exp v0.0.0-20250210185358-939b2ce775ac
	golang.org/x/image v0.32.0
	golang.org/x/net v0.46.0
	golang.org/x/oauth2 v0.32.0
	golang.org/x/sync v0.17.0
	golang.org/x/term v0.36.0
	golang.org/x/text v0.30.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250929231259-57b25ae835d4
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools/v3 v3.5.2
	stash.kopano.io/kgol/rndm v1.1.2
)

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.2 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.5 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/alexedwards/argon2id v1.0.0 // indirect
	github.com/amoghe/go-crypt v0.0.0-20220222110647-20eada5f5964 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.4.3-default-no-op // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/blevesearch/bleve_index_api v1.2.10 // indirect
	github.com/blevesearch/geo v0.2.4 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.12 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.2 // indirect
	github.com/blevesearch/zapx/v12 v12.4.2 // indirect
	github.com/blevesearch/zapx/v13 v13.4.2 // indirect
	github.com/blevesearch/zapx/v14 v14.4.2 // indirect
	github.com/blevesearch/zapx/v15 v15.4.2 // indirect
	github.com/blevesearch/zapx/v16 v16.2.6 // indirect
	github.com/bluele/gcache v0.0.2 // indirect
	github.com/bombsimon/logrusr/v3 v3.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/ceph/go-ceph v0.35.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cevaris/ordered_map v0.0.0-20190319150403-3adeae072e73 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v1.0.0-rc.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/crewjam/httperr v0.2.0 // indirect
	github.com/crewjam/saml v0.4.14 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dgraph-io/ristretto v0.2.0 // indirect
	github.com/dgryski/go-farm v0.0.0-20240924180020-3414d57e47da // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/docker/docker v28.3.3+incompatible // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/egirna/icap v0.0.0-20181108071049-d5ee18bd70bc // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/emvi/iso-639-1 v1.1.1 // indirect
	github.com/evanphx/json-patch/v5 v5.5.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gdexlab/go-render v1.0.1 // indirect
	github.com/go-acme/lego/v4 v4.4.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.8-0.20250403174932-29230038a667 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-git/go-git/v5 v5.13.2 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-micro/plugins/v4/events/natsjs v1.2.2 // indirect
	github.com/go-micro/plugins/v4/store/nats-js v1.2.1 // indirect
	github.com/go-micro/plugins/v4/store/redis v1.2.1 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-resty/resty/v2 v2.7.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.2.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomodule/redigo v1.9.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/renameio/v2 v2.0.0 // indirect
	github.com/gookit/goutil v0.7.1 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/schema v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/juliangruber/go-intersect v1.1.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/kovidgoyal/go-parallel v1.0.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/dsig v1.0.0 // indirect
	github.com/lestrrat-go/dsig-secp256k1 v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.1 // indirect
	github.com/lestrrat-go/jwx/v3 v3.0.11 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/libregraph/oidc-go v1.1.0 // indirect
	github.com/longsleep/go-metrics v1.0.0 // indirect
	github.com/longsleep/rndm v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/maxymania/go-system v0.0.0-20170110133659-647cc364bf0b // indirect
	github.com/mendsley/gojwk v0.0.0-20141217222730-4d5ec6e58103 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/mileusna/useragent v1.3.5 // indirect
	github.com/minio/crc64nvme v1.0.2 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.95 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.8.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/olekukonko/errors v1.1.0 // indirect
	github.com/olekukonko/ll v0.0.9 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pablodz/inotifywaitgo v0.0.9 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/pquerna/cachecontrol v0.2.0 // indirect
	github.com/prometheus/alertmanager v0.28.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/prometheus/statsd_exporter v0.22.8 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/russellhaering/goxmldsig v1.5.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/kafka-go v0.4.49 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/sercand/kuberesolver/v5 v5.1.1 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/sethvargo/go-diceware v0.5.0 // indirect
	github.com/sethvargo/go-password v0.3.1 // indirect
	github.com/shamaton/msgpack/v2 v2.3.1 // indirect
	github.com/shirou/gopsutil/v4 v4.25.6 // indirect
	github.com/shurcooL/httpfs v0.0.0-20230704072500-f1e31cf0ba5c // indirect
	github.com/shurcooL/vfsgen v0.0.0-20230704071429-0000e147ea92 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spacewander/go-suffix-tree v0.0.0-20191010040751-0865e368c784 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/studio-b12/gowebdav v0.9.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/toorop/go-dkim v0.0.0-20201103131630-e1cd1a0a5208 // indirect
	github.com/trustelem/zxcvbn v1.0.1 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/vektah/gqlparser/v2 v2.5.30 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/wk8/go-ordered-map v1.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.etcd.io/etcd/api/v3 v3.6.5 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.5 // indirect
	go.etcd.io/etcd/client/v3 v3.6.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	golang.org/x/tools/godoc v0.1.0-deprecated // indirect
	google.golang.org/genproto v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250929231259-57b25ae835d4 // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace github.com/studio-b12/gowebdav => github.com/kobergj/gowebdav v0.0.0-20250102091030-aa65266db202

replace github.com/unrolled/secure => github.com/DeepDiver1975/secure v0.0.0-20240611112133-abc838fb797c

replace go-micro.dev/v4 => github.com/butonic/go-micro/v4 v4.11.1-0.20241115112658-b5d4de5ed9b3

// exclude the v2 line of go-sqlite3 which was released accidentally and prevents pulling in newer versions of go-sqlite3
// see https://github.com/mattn/go-sqlite3/issues/965 for more details
exclude github.com/mattn/go-sqlite3 v2.0.3+incompatible

replace github.com/go-micro/plugins/v4/store/nats-js-kv => github.com/opencloud-eu/go-micro-plugins/v4/store/nats-js-kv v0.0.0-20250512152754-23325793059a
