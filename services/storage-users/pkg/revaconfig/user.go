package revaconfig

import (
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
)

// StorageProviderDrivers are the drivers for the storage provider
func StorageProviderDrivers(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"eos":          EOS(cfg),
		"eoshome":      EOSHome(cfg),
		"eosgrpc":      EOSGRPC(cfg),
		"local":        Local(cfg),
		"localhome":    LocalHome(cfg),
		"owncloudsql":  OwnCloudSQL(cfg),
		"decomposed":   DecomposedNoEvents(cfg),
		"s3":           S3(cfg),
		"decomposeds3": DecomposedS3NoEvents(cfg),
		"posix":        Posix(cfg, true, cfg.Drivers.Posix.WatchFS),

		"ocis": Decomposed(cfg),           // deprecated: use decomposed
		"s3ng": DecomposedS3NoEvents(cfg), // deprecated: use decomposeds3

	}
}

// DataProviderDrivers are the drivers for the storage provider
func DataProviderDrivers(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"eos":          EOS(cfg),
		"eoshome":      EOSHome(cfg),
		"eosgrpc":      EOSGRPC(cfg),
		"local":        Local(cfg),
		"localhome":    LocalHome(cfg),
		"owncloudsql":  OwnCloudSQL(cfg),
		"decomposed":   Decomposed(cfg),
		"s3":           S3(cfg),
		"decomposeds3": DecomposedS3(cfg),
		"posix":        Posix(cfg, false, false),

		"ocis": Decomposed(cfg),           // deprecated: use decomposed
		"s3ng": DecomposedS3NoEvents(cfg), // deprecated: use decomposeds3
	}
}
