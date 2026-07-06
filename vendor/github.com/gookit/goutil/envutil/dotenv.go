package envutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/goutil/internal/comfunc"
)

// DefaultEnvFile default file name
const DefaultEnvFile = ".env"

// Dotenv load and parse dotenv files
type Dotenv struct {
	// Files dot env file paths, allow multi files.
	//  - filename support simple glob pattern. eg: ".env.*"
	//
	// default: [".env"]
	Files []string
	// BaseDir base dir for join files path
	//
	// default is workdir
	BaseDir string
	// UpperKey change key to upper on set ENV. default: true
	UpperKey bool
	// DotenvFirst overwrite existing os ENV values on load dotenv.
	//
	// default: false - os ENV first
	DotenvFirst bool
	// IgnoreNotExist only load exists.
	//
	// - default: false - will return error if not exists
	IgnoreNotExist bool
	// LoadFirstExist only load first exists env file on Files
	LoadFirstExist bool

	loadFiles []string
	loadData  map[string]string
}

// NewDotenv create a new dotenv config
func NewDotenv() *Dotenv {
	return &Dotenv{
		UpperKey: true,
		Files:    []string{DefaultEnvFile},
		// init fields
		loadData: make(map[string]string),
	}
}

// LoadAndInit load dotenv files and parse to os.Environ
func (c *Dotenv) LoadAndInit() error {
	return c.doLoadFiles(c.Files)
}

// LoadFiles append load dotenv files
//   - filename support simple glob pattern. eg: ".env.*"
func (c *Dotenv) LoadFiles(files ...string) error {
	return c.doLoadFiles(files)
}

// LoadText load dotenv contents and parse to os Env
func (c *Dotenv) LoadText(contents string) error {
	return c.parseAndSetEnv(contents)
}

// do load dotenv files
func (c *Dotenv) doLoadFiles(files []string) error {
	var filePath string
	for _, file := range files {
		filePath = strings.TrimSpace(file)
		if c.BaseDir != "" && !filepath.IsAbs(file) {
			filePath = filepath.Join(c.BaseDir, file)
		}

		// load and parse to ENV
		if err := c.loadFile(filePath); err != nil {
			return err
		}
		if c.LoadFirstExist && len(c.loadFiles) > 0 {
			break
		}
	}

	return nil
}

func (c *Dotenv) loadFile(filePath string) error {
	// filename support simple glob pattern.
	if strings.ContainsRune(filePath, '*') {
		matches, err := filepath.Glob(filePath)
		if err != nil {
			return err
		}

		for _, matchFile := range matches {
			if err = c.parseFile(matchFile); err != nil {
				return err
			}
		}
		return err
	}

	// Load single file
	return c.parseFile(filePath)
}

// parseFile load single file and parse to ENV
func (c *Dotenv) parseFile(filePath string) error {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		// IgnoreNotExist: skip non-existent files
		if c.IgnoreNotExist && os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err = c.parseAndSetEnv(string(contents))
	if err == nil {
		c.loadFiles = append(c.loadFiles, filePath)
	}
	return err
}

func (c *Dotenv) parseAndSetEnv(contents string) error {
	// Parse ENV lines
	envMp, err := comfunc.ParseEnvLines(contents, comfunc.ParseEnvLineOption{
		SkipOnErrorLine: true,
	})

	// Set to ENV
	for key, val := range envMp {
		key = strings.ToUpper(key)
		if !c.DotenvFirst {
			if _, ok := os.LookupEnv(key); ok {
				if _, loaded := c.loadData[key]; !loaded {
					continue
				}
			}
		}
		c.loadData[key] = val
		_ = os.Setenv(key, val)
	}
	return err
}

// UnloadEnv remove loaded dotenv data from os.Environ
func (c *Dotenv) UnloadEnv() bool {
	if len(c.loadData) == 0 {
		return false
	}

	for key := range c.loadData {
		_ = os.Unsetenv(key)
	}
	return true
}

// LoadedData get loaded dotenv data map
func (c *Dotenv) LoadedData() map[string]string {
	return c.loadData
}

// LoadedFiles get loaded dotenv files
func (c *Dotenv) LoadedFiles() []string {
	return c.loadFiles
}

// Reset unload all loaded ENV and reset data
func (c *Dotenv) Reset() {
	c.UnloadEnv()
	c.loadFiles = nil
	c.loadData = make(map[string]string)
}

//
// region standard dotenv instance
//

var stdEnv = NewDotenv()

// StdDotenv get standard dotenv instance
func StdDotenv() *Dotenv { return stdEnv }

// DotenvLoad load dotenv file and parse to ENV
func DotenvLoad(fns ...func(cfg *Dotenv)) error {
	for _, fn := range fns {
		fn(stdEnv)
	}
	return stdEnv.LoadAndInit()
}

// LoadEnvFiles load dotenv files and parse to ENV
func LoadEnvFiles(baseDir string, files ...string) error {
	return DotenvLoad(func(cfg *Dotenv) {
		cfg.BaseDir = baseDir
		if len(files) > 0 {
			cfg.Files = files
		}
	})
}

// LoadedEnvFiles get loaded dotenv files
func LoadedEnvFiles() []string {
	return stdEnv.LoadedFiles()
}
