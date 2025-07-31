package opensearchtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/opencloud-eu/opencloud/services/search/pkg/engine"
)

var Testdata = struct {
	Resources resourceTestdata
}{
	Resources: resourceTestdata{
		Full: loadTestdata[engine.Resource]("resource_full.json"),
	},
}

type resourceTestdata struct {
	Full engine.Resource
}

func loadTestdata[D any](name string) D {
	name = path.Join("internal/test/testdata", name)
	data, err := os.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("failed to read testdata file %s: %v", name, err))
	}

	var d D
	if json.Unmarshal(data, &d) != nil {
		panic(fmt.Sprintf("failed to unmarshal testdata %s: %v", name, err))
	}

	return d
}
