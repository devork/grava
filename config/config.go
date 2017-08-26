package config

import (
	"encoding/json"
	"io"
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

// New will read config from the specified reader
func New(r io.Reader) (*Config, error) {
	cfg := &Config{}
	err := json.NewDecoder(r).Decode(cfg)

	return cfg, err
}
