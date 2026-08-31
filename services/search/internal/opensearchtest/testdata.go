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
		Root:   fromTestData[search.Resource]("resource_root.json"),
		Folder: fromTestData[search.Resource]("resource_folder.json"),
		File:   fromTestData[search.Resource]("resource_file.json"),
	},
}

type resourceTestdata struct {
	Root   search.Resource
	File   search.Resource
	Folder search.Resource
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
