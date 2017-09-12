package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devork/grava/cache"

	"github.com/devork/grava/config"
	"github.com/devork/grava/data"
	"github.com/devork/grava/geo"
	"github.com/devork/grava/web"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

var (
	sha     string
	date    string
	version string
)

var (
	mvtType   = "application/vnd.mapbox-vector-tile"
	protoType = "application/x-protobuf"
	fontsDir  = ""
)

func main() {

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gravad [OPTIONS]\n\nDynamic Mapbox vector tile server for PostGIS")
		fmt.Println()
		flag.PrintDefaults()
	}

	path := flag.String("config", "", "path to `config` file")
	flag.Parse()

	cfg, err := config.New(*path)

	if err != nil {
		log.Errorf("failed to decode configuration: error = %s", err)
		os.Exit(1)
	}

	if cfg.Logging.JSON {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if cfg.Logging.Level != "" {
		lvl, err := log.ParseLevel(cfg.Logging.Level)

		if err != nil {
			log.Errorf("failed to parse level: level = %s, error = %s", cfg.Logging.Level, err)
			os.Exit(1)
		}

		log.SetLevel(lvl)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	db, err := data.NewDb(cfg)
	if err != nil {
		log.Errorf("failed to open database: error = %s", err)
		os.Exit(1)
	}

	defer db.Close()

	fontsDir = cfg.FontsDir
	_, err = os.Open(fontsDir)
	if err != nil {
		log.Errorf("failed to open fonts dir: dir = %s, error = %s", fontsDir, err)
		os.Exit(1)
	}

	c := cache.NewNOOP()
	if cfg.Cache.Type == "memory" {

		if cfg.Cache.Limit <= 0 {
			log.Errorf("Invalid memory cache limit - must be > 0: limit = %d", cfg.Cache.Limit)
			os.Exit(1)
		}
		c = cache.NewMemoryCacher(cfg.Cache.Limit)
	}

	router := mux.NewRouter()
	router.HandleFunc("/status", web.NewStatusHandler("gravad-service"))
	router.HandleFunc("/{name:[A-Za-z0-9_]+}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewErrorHandler(NewMVTHandler(db, c)))
	router.HandleFunc("/fonts/{font}/{file}", web.NewErrorHandler(FontHandler))

	router.NotFoundHandler = http.HandlerFunc(NotFounderHandler)

	log.Infof("starting server: version = %s, build = %s, date = %s, config = %v", version, sha, date, cfg)

	var s *web.Server

	port := cfg.Server.Port

	if port <= 0 {
		port = 8080
	}

	if cfg.Server.CORS {
		s, err = web.NewServer(
			web.NewClacksHandler(web.NewCorsHandler(web.NewRequestHandler("auth-service", router))),
			port,
		)
	} else {
		s, err = web.NewServer(
			web.NewClacksHandler(web.NewRequestHandler("auth-service", router)),
			port,
		)
	}

	if err != nil {
		log.Errorf("failed to create server: error = %s", err)
		os.Exit(1)
	}

	err = s.Run()

	if err != nil {
		log.Errorf("failed to run server: error = %s", err)
		os.Exit(1)
	}

}

// FontHandler will serve static PBF font files
func FontHandler(w http.ResponseWriter, r *http.Request) *web.Error {
	vars := mux.Vars(r)
	font := vars["font"]
	file := vars["file"]

	if !strings.HasSuffix(file, "pbf") {
		return &web.Error{
			Status:  http.StatusNotFound,
			Code:    0,
			Message: "No such file",
		}
	}

	fontFile, err := os.Open(filepath.Join(fontsDir, font, file))

	if err != nil {
		log.Errorf("failed to open font path: requested font = %s, file = %s, error = %s", font, file, err)
		return &web.Error{
			Status:  http.StatusInternalServerError,
			Code:    0,
			Message: "Failed to open font file",
		}
	}

	defer fontFile.Close()

	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, fontFile)

	if err != nil {
		log.Errorf("failed to open font path: requested font = %s, file = %s, error = %s", font, file, err)
		return &web.Error{
			Status:  http.StatusInternalServerError,
			Code:    0,
			Message: "Failed to open font file",
		}
	}

	return nil

}

// NewMVTHandler will create a handler function that is responsible for handling all requests for vector tiles.
func NewMVTHandler(db *data.Db, cache cache.Cacher) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) *web.Error {

		vars := mux.Vars(r)
		x, _ := strconv.Atoi(vars["x"])
		y, _ := strconv.Atoi(vars["y"])
		z, _ := strconv.Atoi(vars["z"])
		name := vars["name"]

		key := fmt.Sprintf("%s_%s_%s_%s", name, vars["x"], vars["y"], vars["z"])

		var data []byte
		data, err := cache.Get(key)

		if err != nil {
			log.Errorf("Cache fetch failed: key = %s, error = %s", key, err)
		}

		if data != nil {
			log.Debugf("cache tile fetched: key = %s", key)
			w.Header().Add("Content-Type", mvtType)
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusOK)
			w.Write(data)

			return nil
		}

		box := geo.NewBBox(x, y, z)

		// get data from bbox
		tile, err := db.FetchTile(box, name)

		if err != nil {
			log.Errorf("failed to perform tile query: error = %s", err)
			return &web.Error{
				Status:  http.StatusInternalServerError,
				Code:    0,
				Message: "failed to query data",
			}
		}

		data, err = proto.Marshal(tile)

		if err != nil {
			log.Errorf("failed to marshal tile to protobuf: error = %s", err)
			return &web.Error{
				Status:  http.StatusInternalServerError,
				Code:    0,
				Message: "failed to marshal tile to protobuf",
			}
		}

		cache.Set(key, data)

		w.Header().Add("Content-Type", mvtType)
		w.Header().Add("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)

		return nil
	}
}

// NotFounderHandler provides extra logging when no route matches
func NotFounderHandler(w http.ResponseWriter, r *http.Request) {
	log.Warnf("cannot find handler for route: route = %s", r.RequestURI)
	w.WriteHeader(http.StatusNotFound)
}
