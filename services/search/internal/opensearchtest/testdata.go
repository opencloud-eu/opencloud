package opensearchtest

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

//go:embed testdata/*.json
var testdata embed.FS

var Testdata = struct {
	Resources resourceTestdata
}{
	Resources: resourceTestdata{
		File: fromTestData[search.Resource]("resource_file.json"),
	},
}

type resourceTestdata struct {
	File search.Resource
}

func fromTestData[D any](name string) D {
	name = path.Join("./testdata", name)
	data, err := testdata.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("failed to read testdata file %s: %v", name, err))
	}

	var d D
	if json.Unmarshal(data, &d) != nil {
		panic(fmt.Sprintf("failed to unmarshal testdata %s: %v", name, err))
	}

	return d
}
