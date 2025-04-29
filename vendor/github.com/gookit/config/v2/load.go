package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/fsutil"
)

// LoadFiles load one or multi files, will fire OnLoadData event
//
// Usage:
//
//	config.LoadFiles(file1, file2, ...)
func LoadFiles(sourceFiles ...string) error { return dc.LoadFiles(sourceFiles...) }

// LoadFiles load and parse config files, will fire OnLoadData event
func (c *Config) LoadFiles(sourceFiles ...string) (err error) {
	for _, file := range sourceFiles {
		if err = c.loadFile(file, false, ""); err != nil {
			return
		}
	}
	return
}

// LoadExists load one or multi files, will ignore not exist
//
// Usage:
//
//	config.LoadExists(file1, file2, ...)
func LoadExists(sourceFiles ...string) error { return dc.LoadExists(sourceFiles...) }

// LoadExists load and parse config files, but will ignore not exists file.
func (c *Config) LoadExists(sourceFiles ...string) (err error) {
	for _, file := range sourceFiles {
		if file == "" {
			continue
		}

		if err = c.loadFile(file, true, ""); err != nil {
			return
		}
	}
	return
}

// LoadRemote load config data from remote URL.
func LoadRemote(format, url string) error { return dc.LoadRemote(format, url) }

// LoadRemote load config data from remote URL.
//
// Usage:
//
//	c.LoadRemote(config.JSON, "http://abc.com/api-config.json")
func (c *Config) LoadRemote(format, url string) (err error) {
	// create http client
	client := http.Client{Timeout: 300 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	//noinspection GoUnhandledErrorResult
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("fetch remote config error, reply status code is %d", resp.StatusCode)
	}

	// read response content
	bts, err := io.ReadAll(resp.Body)
	if err == nil {
		if err = c.parseSourceCode(format, bts); err != nil {
			return
		}
		c.loadedUrls = append(c.loadedUrls, url)
	}
	return
}

// LoadOSEnv load data from OS ENV
//
// Deprecated: please use LoadOSEnvs()
func LoadOSEnv(keys []string, keyToLower bool) { dc.LoadOSEnv(keys, keyToLower) }

// LoadOSEnv load data from os ENV
//
// Deprecated: please use Config.LoadOSEnvs()
func (c *Config) LoadOSEnv(keys []string, keyToLower bool) {
	for _, key := range keys {
		// NOTICE: if is Windows os, os.Getenv() Key is not case-sensitive
		val := os.Getenv(key)
		if keyToLower {
			key = strings.ToLower(key)
		}
		_ = c.Set(key, val)
	}
	c.fireHook(OnLoadData)
}

// LoadOSEnvs load data from OS ENVs. see Config.LoadOSEnvs
func LoadOSEnvs(nameToKeyMap map[string]string) { dc.LoadOSEnvs(nameToKeyMap) }

// LoadOSEnvs load data from os ENVs. format: `{ENV_NAME: config_key}`
//
//   - `config_key` allow use key path. eg: `{"DB_USERNAME": "db.username"}`
func (c *Config) LoadOSEnvs(nameToKeyMap map[string]string) {
	for name, cfgKey := range nameToKeyMap {
		if val := os.Getenv(name); val != "" {
			if cfgKey == "" {
				cfgKey = strings.ToLower(name)
			}
			_ = c.Set(cfgKey, val)
		}
	}

	c.fireHook(OnLoadData)
}

// support bound types for CLI flags vars
var validTypes = map[string]int{
	"int":  1,
	"uint": 1,
	"bool": 1,
	// string is default
	"string": 1,
}

// LoadFlags load data from cli flags. see Config.LoadFlags
func LoadFlags(defines []string) error { return dc.LoadFlags(defines) }

// LoadFlags parse command line arguments, based on provide keys.
//
// Usage:
//
//	// 'debug' flag is bool type
//	c.LoadFlags([]string{"env", "debug:bool"})
//	// can with flag desc message
//	c.LoadFlags([]string{"env:set the run env"})
//	c.LoadFlags([]string{"debug:bool:set debug mode"})
//	// can set value to map key. eg: myapp --map1.sub-key=val
//	c.LoadFlags([]string{"--map1.sub-key"})
func (c *Config) LoadFlags(defines []string) (err error) {
	hash := map[string]int8{}

	// bind vars
	for _, str := range defines {
		key, typ, desc := parseVarNameAndType(str)
		if desc == "" {
			desc = "config flag " + key
		}

		switch typ {
		case "int":
			ptr := new(int)
			flag.IntVar(ptr, key, c.Int(key), desc)
			hash[key] = 0
		case "uint":
			ptr := new(uint)
			flag.UintVar(ptr, key, c.Uint(key), desc)
			hash[key] = 0
		case "bool":
			ptr := new(bool)
			flag.BoolVar(ptr, key, c.Bool(key), desc)
			hash[key] = 0
		default: // as string
			ptr := new(string)
			flag.StringVar(ptr, key, c.String(key), desc)
			hash[key] = 0
		}
	}

	// parse and collect
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		name := f.Name
		// only get name in the keys.
		if _, ok := hash[name]; !ok {
			return
		}

		// if f.Value implement the flag.Getter, read typed value
		if gtr, ok := f.Value.(flag.Getter); ok {
			_ = c.Set(name, gtr.Get())
			// } else { // TIP: basic type flag always implements Getter interface
			// 	_ = c.Set(name, f.Value.String()) // ignore error
		}
	})

	c.fireHook(OnLoadData)
	return
}

// LoadData load one or multi data
func LoadData(dataSource ...any) error { return dc.LoadData(dataSource...) }

// LoadData load data from map OR struct
//
// The dataSources type allow:
//   - map[string]any
//   - map[string]string
func (c *Config) LoadData(dataSources ...any) (err error) {
	if c.opts.Delimiter == 0 {
		c.opts.Delimiter = defaultDelimiter
	}

	var loaded bool
	for _, ds := range dataSources {
		if smp, ok := ds.(map[string]string); ok {
			loaded = true
			c.LoadSMap(smp)
			continue
		}

		err = mergo.Merge(&c.data, ds, c.opts.MergeOptions...)
		if err != nil {
			return errorx.WithStack(err)
		}
		loaded = true
	}

	if loaded {
		c.fireHook(OnLoadData)
	}
	return
}

// LoadSMap to config
func (c *Config) LoadSMap(smp map[string]string) {
	for k, v := range smp {
		c.data[k] = v
	}
	c.fireHook(OnLoadData)
}

// LoadSources load one or multi byte data
func LoadSources(format string, src []byte, more ...[]byte) error {
	return dc.LoadSources(format, src, more...)
}

// LoadSources load data from byte content.
//
// Usage:
//
//	config.LoadSources(config.Yaml, []byte(`
//	  name: blog
//	  arr:
//		key: val
//
// `))
func (c *Config) LoadSources(format string, src []byte, more ...[]byte) (err error) {
	err = c.parseSourceCode(format, src)
	if err != nil {
		return
	}

	for _, sc := range more {
		err = c.parseSourceCode(format, sc)
		if err != nil {
			return
		}
	}
	return
}

// LoadStrings load one or multi string
func LoadStrings(format string, str string, more ...string) error {
	return dc.LoadStrings(format, str, more...)
}

// LoadStrings load data from source string content.
func (c *Config) LoadStrings(format string, str string, more ...string) (err error) {
	err = c.parseSourceCode(format, []byte(str))
	if err != nil {
		return
	}

	for _, s := range more {
		err = c.parseSourceCode(format, []byte(s))
		if err != nil {
			return
		}
	}
	return
}

// LoadFilesByFormat load one or multi config files by give format, will fire OnLoadData event
func LoadFilesByFormat(format string, configFiles ...string) error {
	return dc.LoadFilesByFormat(format, configFiles...)
}

// LoadFilesByFormat load one or multi files by give format, will fire OnLoadData event
func (c *Config) LoadFilesByFormat(format string, configFiles ...string) (err error) {
	for _, file := range configFiles {
		if err = c.loadFile(file, false, format); err != nil {
			return
		}
	}
	return
}

// LoadExistsByFormat load one or multi files by give format, will fire OnLoadData event
func LoadExistsByFormat(format string, configFiles ...string) error {
	return dc.LoadExistsByFormat(format, configFiles...)
}

// LoadExistsByFormat load one or multi files by give format, will fire OnLoadData event
func (c *Config) LoadExistsByFormat(format string, configFiles ...string) (err error) {
	for _, file := range configFiles {
		if err = c.loadFile(file, true, format); err != nil {
			return
		}
	}
	return
}

// LoadOptions for load config from dir.
type LoadOptions struct {
	// DataKey use for load config from dir.
	// see https://github.com/gookit/config/issues/173
	DataKey string
}

// LoadOptFn type func
type LoadOptFn func(lo *LoadOptions)

func newLoadOptions(loFns []LoadOptFn) *LoadOptions {
	lo := &LoadOptions{}
	for _, fn := range loFns {
		fn(lo)
	}
	return lo
}

// LoadFromDir Load custom format files from the given directory, the file name will be used as the key.
//
// Example:
//
//	// file: /somedir/task.json
//	LoadFromDir("/somedir", "json")
//
//	// after load
//	Config.data = map[string]any{"task": file data}
func LoadFromDir(dirPath, format string, loFns ...LoadOptFn) error {
	return dc.LoadFromDir(dirPath, format, loFns...)
}

// LoadFromDir Load custom format files from the given directory, the file name will be used as the key.
//
// NOTE: will not be reloaded on call ReloadFiles(), if data loaded by the method.
//
// Example:
//
//	// file: /somedir/task.json , will use filename 'task' as key
//	Config.LoadFromDir("/somedir", "json")
//
//	// after load, the data will be:
//	Config.data = map[string]any{"task": {file data}}
func (c *Config) LoadFromDir(dirPath, format string, loFns ...LoadOptFn) (err error) {
	extName := "." + format
	extLen := len(extName)

	lo := newLoadOptions(loFns)
	dirData := make(map[string]any)
	dataList := make([]map[string]any, 0, 8)

	err = fsutil.FindInDir(dirPath, func(fPath string, ent fs.DirEntry) error {
		baseName := ent.Name()
		if strings.HasSuffix(baseName, extName) {
			data, err := c.parseSourceToMap(format, fsutil.MustReadFile(fPath))
			if err != nil {
				return err
			}

			// filename without ext.
			onlyName := baseName[:len(baseName)-extLen]
			if lo.DataKey != "" {
				dataList = append(dataList, data)
			} else {
				dirData[onlyName] = data
			}

			// TODO use file name as key, it cannot be reloaded. So, cannot append to loadedFiles
			// c.loadedFiles = append(c.loadedFiles, fPath)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if lo.DataKey != "" {
		dirData[lo.DataKey] = dataList
	}

	if len(dirData) == 0 {
		return nil
	}
	return c.loadDataMap(dirData)
}

// ReloadFiles reload config data use loaded files
func ReloadFiles() error { return dc.ReloadFiles() }

// ReloadFiles reload config data use loaded files. use on watching loaded files change
func (c *Config) ReloadFiles() (err error) {
	files := c.loadedFiles
	if len(files) == 0 {
		return
	}

	data := c.Data()
	c.reloading = true
	c.ClearCaches()

	defer func() {
		// revert to back up data on error
		if err != nil {
			c.data = data
		}

		c.lock.Unlock()
		c.reloading = false

		if err == nil {
			c.fireHook(OnReloadData)
		}
	}()

	// with lock
	c.lock.Lock()

	// reload config files
	return c.LoadFiles(files...)
}

// load config file, will fire OnLoadData event
func (c *Config) loadFile(file string, loadExist bool, format string) (err error) {
	fd, err := os.Open(file)
	if err != nil {
		// skip not exist file
		if os.IsNotExist(err) && loadExist {
			return nil
		}
		return err
	}
	//noinspection GoUnhandledErrorResult
	defer fd.Close()

	// read file content
	bts, err := io.ReadAll(fd)
	if err == nil {
		// get format for file ext
		if format == "" {
			format = strings.Trim(filepath.Ext(file), ".")
		}

		// parse file content
		if err = c.parseSourceCode(format, bts); err != nil {
			return
		}

		if !c.reloading {
			c.loadedFiles = append(c.loadedFiles, file)
		}
	}
	return
}

// parse config source code to Config.
func (c *Config) parseSourceCode(format string, blob []byte) (err error) {
	data, err := c.parseSourceToMap(format, blob)
	if err != nil {
		return err
	}

	return c.loadDataMap(data)
}

func (c *Config) loadDataMap(data map[string]any) (err error) {
	// first: init config data
	if len(c.data) == 0 {
		c.data = data
	} else {
		// again ... will merge data
		err = mergo.Merge(&c.data, data, c.opts.MergeOptions...)
	}

	if !c.reloading && err == nil {
		c.fireHook(OnLoadData)
	}
	return err
}

// parse config source code to Config.
func (c *Config) parseSourceToMap(format string, blob []byte) (map[string]any, error) {
	format = c.resolveFormat(format)
	decode := c.decoders[format]
	if decode == nil {
		return nil, errors.New("not register decoder for the format: " + format)
	}

	if c.opts.Delimiter == 0 {
		c.opts.Delimiter = defaultDelimiter
	}

	// decode content to data
	data := make(map[string]any)

	if err := decode(blob, &data); err != nil {
		return nil, err
	}
	return data, nil
}
