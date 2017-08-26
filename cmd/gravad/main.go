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

	"github.com/devork/grava/config"
	"github.com/devork/grava/data"
	"github.com/devork/grava/geo"
	"github.com/devork/grava/web"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

// build flags
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

var (
	rprof interface {
		Stop()
	}
)

func main() {

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gravad [OPTIONS]\n\nDynamic Mapbox vector tile server for PostGIS")
		fmt.Println()
		flag.PrintDefaults()
	}

	// configure available layers etc
	path := flag.String("config", "", "path to `config` file")
	port := flag.Int("port", 8080, "`port` on which the service will listen")
	cors := flag.Bool("with-cors", false, "enable or disable CORS on requests (turn off when behind HAProxy for example)")
	flag.Parse()

	cfg, err := config.New(*path)

	if err != nil {
		log.Errorf("failed to decode configuration: error = %s", err)
		os.Exit(1)
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

	router := mux.NewRouter()
	router.HandleFunc("/status", web.NewStatusHandler("gravad-service"))
	router.HandleFunc("/{name:[A-Za-z0-9_]+}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewErrorHandler(NewMVTHandler(db)))
	router.HandleFunc("/fonts/{font}/{file}", web.NewErrorHandler(FontHandler))

	router.NotFoundHandler = http.HandlerFunc(NotFounderHandler)

	log.Infof("starting server: build = %s, date = %s, config = %v", sha, date, cfg)

	var s *web.Server

	if *cors {
		s, err = web.NewServer(web.NewCorsHandler(web.NewRequestHandler("auth-service", router)), *port)
	} else {
		s, err = web.NewServer(web.NewRequestHandler("auth-service", router), *port)
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
func NewMVTHandler(db *data.Db) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) *web.Error {

		vars := mux.Vars(r)
		log.Info("handling request", "layer", vars["layer"], "x", vars["x"], "y", vars["y"], "z", vars["z"])
		x, _ := strconv.Atoi(vars["x"])
		y, _ := strconv.Atoi(vars["y"])
		z, _ := strconv.Atoi(vars["z"])
		name := vars["name"]

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

		data, err := proto.Marshal(tile)

		if err != nil {
			log.Errorf("failed to marshal tile to protobuf: error = %s", err)
			return &web.Error{
				Status:  http.StatusInternalServerError,
				Code:    0,
				Message: "failed to marshal tile to protobuf",
			}
		}

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
