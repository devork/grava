package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bitbucket.org/devork/kjore/log"
)

// Config is the parsed configuration file
//
// Postgis Configuration
//
// Postgres connection string can either be a URI or key/value pairs - for more details see:
//
//      http://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
//
// Sample PostGIS
//
// "postgres": "postgresql://user:password@/mvt"
//
type Config struct {
	Postgres string   `json:"postgres"`
	Schema   string   `json:"schema"`
	Sources  []Source `json:"sources"`
	FontsDir string   `json:"fontsDir"`
	Path     string   `json:"-"`
}

// Source configures a set of layers to be displayed in a vector map. A source is composed of a name (which must be unique in the set of
// sources configured), a URI (which points to a datasource) and the set of layers to query.
//
// Postgis Database:
//
//  {
//      "name": "opmplc_su",
//      "layers": [
//          "namedplace",
//          "building"
//      ]
//  }
//
type Source struct {
	Prefix string   `json:"prefix"`
	Name   string   `json:"name"`
	Layers []string `json:"layers"`
}

// New will read config from the specified path
func New(path string) (*Config, error) {
	cfg := &Config{}

	p, err := filepath.Abs(path)

	if err != nil {
		log.Errorf("failed to resolve path: error = %s", err)
		os.Exit(1)
	}

	file, err := os.Open(p)

	if err != nil {
		log.Errorf("failed to open config path: path = %s, error = %s", path, err)
		os.Exit(1)
	}

	err = json.NewDecoder(file).Decode(cfg)

	if err != nil {
		return nil, err
	}

	if cfg.FontsDir == "" || !filepath.IsAbs(cfg.FontsDir) {

		base := filepath.Dir(p)
		fontsPath := filepath.Join(base, cfg.FontsDir)
		fontsPath, err = filepath.Abs(fontsPath)

		if err != nil {
			return nil, fmt.Errorf("failed to resolve font path: error = %s", err)
		}

		finfo, _ := os.Lstat(fontsPath)

		if !finfo.IsDir() {
			return nil, fmt.Errorf("font path is not a directory: path = %s", fontsPath)
		}

		cfg.FontsDir = fontsPath
	}

	cfg.Path = path

	return cfg, err
}
