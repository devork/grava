package data

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/devork/grava/config"
	"github.com/devork/grava/geo"
	"github.com/devork/grava/vtile"

	"github.com/devork/geom"
	"github.com/devork/geom/ewkb"
	"github.com/jackc/pgx"

	log "github.com/sirupsen/logrus"
)

var (
	// TODO: make this configurable
	tileSize uint32 = 4096

	// vector version:
	mvtVersion = uint32(2)

	point      = vtile.Tile_POINT
	polygon    = vtile.Tile_POLYGON
	linestring = vtile.Tile_LINESTRING
)

// Common errors
var (
	ErrNoSuchSource = errors.New("no such source with given name")
)

// Db holds the Database connection
type Db struct {
	db      pgx.ConnPool
	sources map[string][]*Layer
}

// Close will release all resources associated with the database
func (d *Db) Close() {
	d.db.Close()
}

// FetchTile queries the database for those features which intersect the given BBOX and the specified layer(s)
func (d *Db) FetchTile(box *geo.BBox, name string) (*vtile.Tile, error) {

	layers, ok := d.sources[name]

	if !ok {
		return nil, ErrNoSuchSource
	}

	// tile pixel width in ground units
	width := float64(tileSize) / (box.Maxx - box.Minx)
	height := float64(tileSize) / (box.Maxy - box.Miny)

	// next tile instance
	tile := vtile.Tile{}
	tile.Layers = []*vtile.Tile_Layer{}
	var vlayer *vtile.Tile_Layer
	var err error

	log.Debugf("Fetching tile data: bbox = %s, name = %s, width = %f, height = %f", box.GoString(), name, width, height)
	for _, layer := range layers {
		vlayer, err = d.readLayer(layer, box, width, height)

		if err != nil {
			return nil, err
		}
		tile.Layers = append(tile.Layers, vlayer)
	}

	return &tile, nil

}

func (d *Db) readLayer(lyr *Layer, box *geo.BBox, width, height float64) (*vtile.Tile_Layer, error) {
	bx := (box.Maxx - box.Minx) * 0.05
	by := (box.Maxy - box.Miny) * 0.05
	log.Debug("Reading Layer: bbox = %s, name = %s, bx = %f, by = %f", box.GoString(), lyr.Name, bx, by)
	rows, err := d.db.Query(
		lyr.Query,
		box.Minx-bx, box.Miny-by, box.Maxx+bx, box.Maxy+by, box.Srid,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	desc := rows.FieldDescriptions()

	// key -> index
	skips := []int{}
	keys := make([]string, len(desc))
	for idx, key := range desc {

		if key.Name == "geom" || key.Name == "geometry" {
			skips = append(skips, idx)
		}
		keys[idx] = key.Name
	}

	// value -> key index
	values := map[interface{}]int{}
	vcount := 0

	var ins []interface{}

	var tileLayer = vtile.Tile_Layer{}
	tileLayer.Version = &mvtVersion
	tileLayer.Name = &lyr.Name
	tileLayer.Extent = &tileSize

	var features = make([]*vtile.Tile_Feature, 0)
	var r io.Reader

	for rows.Next() {
		ins, err = rows.Values()
		if err != nil {
			return nil, err
		}

		r = bytes.NewReader(ins[0].([]byte))
		g, err := ewkb.Decode(r)

		if err != nil {
			return nil, err
		}

		var feature *vtile.Tile_Feature

		// TODO: read multipoint
		switch g.(type) {
		case *geom.Point:
			feature = readPoint(g.(*geom.Point), box, width, height)
		case *geom.Polygon:
			feature = readPolygon(g.(*geom.Polygon), box, width, height)
		case *geom.MultiPolygon:
			feature = readMultiPolygon(g.(*geom.MultiPolygon), box, width, height)
		case *geom.LineString:
			feature = readLinestring(g.(*geom.LineString), box, width, height)
		case *geom.MultiLineString:
			feature = readMultiLinestring(g.(*geom.MultiLineString), box, width, height)
		default:
			log.Warn("unsupported geometry type", "geometry", g.Type())
			continue
		}

	vloop:
		for idx, value := range ins {

			for _, skip := range skips {
				if idx == skip {
					continue vloop
				}
			}

			if value == nil {
				log.Debugf("found nil value: position = %d, column = %s", idx, desc[idx].Name)
				continue
			}

			if v, ok := value.([]string); ok {
				value = strings.Join(v, ",")
			}

			valuesIdx, ok := values[value]

			if !ok {
				values[value] = vcount
				valuesIdx = vcount
				vcount++
			}

			feature.Tags = append(feature.Tags, uint32(idx), uint32(valuesIdx))
		}

		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	tileLayer.Features = features
	tileLayer.Keys = keys
	tileLayer.Values = make([]*vtile.Tile_Value, len(values))

	// create values
	for v, i := range values {
		switch v.(type) {
		case string:
			var str = v.(string)
			tileLayer.Values[i] = &vtile.Tile_Value{StringValue: &str}
		case float32:
			var f32 = v.(float32)
			tileLayer.Values[i] = &vtile.Tile_Value{FloatValue: &f32}
		case float64:
			var f64 = v.(float64)
			tileLayer.Values[i] = &vtile.Tile_Value{DoubleValue: &f64}
		case int:
			var i32 = int64(v.(int))
			tileLayer.Values[i] = &vtile.Tile_Value{IntValue: &i32}
		case int32:
			var i32 = int64(v.(int32))
			tileLayer.Values[i] = &vtile.Tile_Value{IntValue: &i32}
		case uint, uint32, uint64:
			var u64 = v.(uint64)
			tileLayer.Values[i] = &vtile.Tile_Value{UintValue: &u64}
		case int64:
			var i64 = v.(int64)
			tileLayer.Values[i] = &vtile.Tile_Value{SintValue: &i64}
		case bool:
			var b = v.(bool)
			tileLayer.Values[i] = &vtile.Tile_Value{BoolValue: &b}
		case time.Time:
			str := fmt.Sprint(v)
			tileLayer.Values[i] = &vtile.Tile_Value{StringValue: &str}
		default:
			str := fmt.Sprint(v)
			tileLayer.Values[i] = &vtile.Tile_Value{StringValue: &str}
			log.Warn("unexpected type", "value", v, "type", fmt.Sprintf("%T", v))
		}
	}

	return &tileLayer, nil
}

func readPoint(g *geom.Point, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	feature.Type = &point

	// convert to coords
	gx := g.Coordinate[0]
	gy := g.Coordinate[1]

	x := int32(math.Floor((gx - box.Minx) * width))
	y := int32(math.Floor((gy - box.Miny) * height))

	cmd := uint32(1&0x7 | 1<<3)
	px := uint32(x<<1 ^ x>>31)
	py := uint32(y<<1 ^ y>>31)
	feature.Geometry = []uint32{cmd, px, py}

	return &feature
}

func readMultiPoint(mp *geom.MultiPoint, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	feature.Type = &point

	for _, g := range mp.Points {

		// convert to coords
		gx := g.Coordinate[0]
		gy := g.Coordinate[1]

		x := int32(math.Floor((gx - box.Minx) * width))
		y := int32(math.Floor((gy - box.Miny) * height))

		cmd := uint32(1&0x7 | 1<<3)
		px := uint32(x<<1 ^ x>>31)
		py := uint32(y<<1 ^ y>>31)
		feature.Geometry = []uint32{cmd, px, py}
	}

	return &feature
}

func readLinestring(g *geom.LineString, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	var gx float64
	var gy float64
	var dx int32
	var dy int32
	var cmds = []uint32{}
	var x int32
	var y int32
	var lx int32
	var ly int32

	// moveto
	gx = g.Coordinates[0][0]
	gy = g.Coordinates[0][1]

	x = int32(math.Floor((gx - box.Minx) * width))
	y = int32(math.Floor((gy - box.Miny) * height))

	dx = x - lx
	dy = y - ly

	cmd := uint32(1&0x7 | 1<<3)
	cmds = append(cmds, cmd, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))

	lx = x
	ly = y

	// lineto
	// (-1) as we have consumed the first point
	cmd = uint32(2&0x07 | (len(g.Coordinates)-1)<<3)
	cmds = append(cmds, cmd)

	for idx := 1; idx < len(g.Coordinates); idx++ {
		gx = g.Coordinates[idx][0]
		gy = g.Coordinates[idx][1]

		x = int32(math.Floor((gx - box.Minx) * width))
		y = int32(math.Floor((gy - box.Miny) * height))

		dx = x - lx
		dy = y - ly

		lx = x
		ly = y

		cmds = append(cmds, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))
	}

	feature.Geometry = cmds
	feature.Type = &linestring

	return &feature
}

func readMultiLinestring(mls *geom.MultiLineString, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	var gx float64
	var gy float64
	var dx int32
	var dy int32
	var cmds = []uint32{}
	var x int32
	var y int32
	var lx int32
	var ly int32

	for _, g := range mls.LineStrings {

		// moveto
		gx = g.Coordinates[0][0]
		gy = g.Coordinates[0][1]

		x = int32(math.Floor((gx - box.Minx) * width))
		y = int32(math.Floor((gy - box.Miny) * height))

		dx = x - lx
		dy = y - ly

		cmd := uint32(1&0x7 | 1<<3)
		cmds = append(cmds, cmd, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))

		lx = x
		ly = y

		// lineto
		// (-1) as we have consumed the first point
		cmd = uint32(2&0x07 | (len(g.Coordinates)-1)<<3)
		cmds = append(cmds, cmd)

		for idx := 1; idx < len(g.Coordinates); idx++ {
			gx = g.Coordinates[idx][0]
			gy = g.Coordinates[idx][1]

			x = int32(math.Floor((gx - box.Minx) * width))
			y = int32(math.Floor((gy - box.Miny) * height))

			dx = x - lx
			dy = y - ly

			lx = x
			ly = y

			cmds = append(cmds, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))
		}
	}

	feature.Geometry = cmds
	feature.Type = &linestring

	return &feature
}

func readPolygon(g *geom.Polygon, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	var gx float64
	var gy float64
	var dx int32
	var dy int32
	var cmds = []uint32{}
	var x int32
	var y int32
	var lx int32
	var ly int32

	for _, ring := range g.Rings {
		// moveto
		gx = ring.Coordinates[0][0]
		gy = ring.Coordinates[0][1]

		x = int32(math.Floor((gx - box.Minx) * width))
		y = int32(math.Floor((gy - box.Miny) * height))

		dx = x - lx
		dy = y - ly

		cmd := uint32(1&0x7 | 1<<3)
		cmds = append(cmds, cmd, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))

		//log.Printf("moveto: %d %d => %d %d", x, y, dx, dy)

		lx = x
		ly = y

		// lineto
		// (-2) as we have consumed the first point and ignoring the final point in the ring
		cmd = uint32(2&0x07 | (len(ring.Coordinates)-2)<<3)
		cmds = append(cmds, cmd)

		// ignore last coord as that is a close path
		for idx := 1; idx < len(ring.Coordinates)-1; idx++ {
			gx = ring.Coordinates[idx][0]
			gy = ring.Coordinates[idx][1]

			x = int32(math.Floor((gx - box.Minx) * width))
			y = int32(math.Floor((gy - box.Miny) * height))

			dx = x - lx
			dy = y - ly

			lx = x
			ly = y

			cmds = append(cmds, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))
		}

		//closepath
		cmd = uint32(7&0x07 | 1<<3)
		cmds = append(cmds, cmd)
	}

	feature.Geometry = cmds
	feature.Type = &polygon

	return &feature
}

func readMultiPolygon(g *geom.MultiPolygon, box *geo.BBox, width, height float64) *vtile.Tile_Feature {
	var feature = vtile.Tile_Feature{}
	var gx float64
	var gy float64
	var dx int32
	var dy int32
	var cmds = []uint32{}
	var x int32
	var y int32
	var lx int32
	var ly int32

	for _, poly := range g.Polygons {

		for _, ring := range poly.Rings {
			// moveto
			gx = ring.Coordinates[0][0]
			gy = ring.Coordinates[0][1]

			x = int32(math.Floor((gx - box.Minx) * width))
			y = int32(math.Floor((gy - box.Miny) * height))

			dx = x - lx
			dy = y - ly

			cmd := uint32(1&0x7 | 1<<3)
			cmds = append(cmds, cmd, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))

			//log.Printf("moveto: %d %d => %d %d", x, y, dx, dy)

			lx = x
			ly = y

			// lineto
			// (-2) as we have consumed the first point and ignoring the final point in the ring
			cmd = uint32(2&0x07 | (len(ring.Coordinates)-2)<<3)
			cmds = append(cmds, cmd)

			// ignore last coord as that is a close path
			for idx := 1; idx < len(ring.Coordinates)-1; idx++ {
				gx = ring.Coordinates[idx][0]
				gy = ring.Coordinates[idx][1]

				x = int32(math.Floor((gx - box.Minx) * width))
				y = int32(math.Floor((gy - box.Miny) * height))

				dx = x - lx
				dy = y - ly

				lx = x
				ly = y

				cmds = append(cmds, uint32(dx<<1^dx>>31), uint32(dy<<1^dy>>31))
			}

			//close path
			cmd = uint32(7&0x07 | 1<<3)
			cmds = append(cmds, cmd)
		}
	}

	feature.Geometry = cmds
	feature.Type = &polygon

	return &feature
}

// NewDb opens the database specified at the given path
func NewDb(cfg *config.Config) (*Db, error) {

	if cfg.Postgres == "" {
		return nil, errors.New("no postgres configuration specified")
	}

	var pcon pgx.ConnConfig
	var err error

	if strings.HasPrefix(cfg.Postgres, "postgres://") || strings.HasPrefix(cfg.Postgres, "postgresql://") {
		pcon, err = pgx.ParseURI(cfg.Postgres)

		if err != nil {
			return nil, fmt.Errorf("failed to parse connection uri: %s", err)
		}
	} else {
		pcon, err = pgx.ParseDSN(cfg.Postgres)

		if err != nil {
			return nil, fmt.Errorf("failed to parse connection DSN: %s", err)
		}
	}

	pcon.Logger = &logger{}
	pcon.LogLevel = pgx.LogLevelError

	db, err := pgx.NewConnPool(
		pgx.ConnPoolConfig{
			ConnConfig:     pcon,
			MaxConnections: 10,
		},
	)

	if err != nil {
		return nil, err
	}

	sources := map[string][]*Layer{}
	for _, source := range cfg.Sources {
		sources[source.Name] = make([]*Layer, len(source.Layers))

		for idx, lyr := range source.Layers {
			sources[source.Name][idx], err = read(db, source.Prefix, lyr, cfg.Schema)

			if err != nil {
				return nil, err
			}
		}
	}

	return &Db{*db, sources}, nil
}

func read(db *pgx.ConnPool, prefix, layer, schema string) (*Layer, error) {
	rows, err := db.Query(`
		select 
			column_name, udt_name 
		from 
			information_schema.columns 
		where 
			table_schema = $1 and table_name = $2 
		order by 
			ordinal_position asc
		`, schema, prefix+layer,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	cols := []string{}
	var col string
	var udt string
	var geom string
	var columns string

	count := 0
	for rows.Next() {
		err = rows.Scan(&col, &udt)

		if err != nil {
			return nil, err
		}

		count++

		switch udt {
		case "int2", "int4", "int8", "float4", "float8", "bool", "varchar", "text":
			cols = append(cols, col)
		case "geometry":
			geom = col
		}
	}

	if count == 0 {
		return nil, fmt.Errorf("no table found: name = %s", layer)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if geom == "" {
		return nil, fmt.Errorf("no geometry column found: table = %s", layer)
	}

	if len(cols) > 0 {
		columns = ", " + strings.Join(cols, ",")
	}

	return &Layer{
		layer,
		fmt.Sprintf(
			`select 
				ST_AsBinary(ST_Intersection(%s, st_makeenvelope($1, $2, $3, $4, $5))) as geom %s 
			from 
				%s.%s%s 
			where 
				st_intersects(geometry, st_makeenvelope($1, $2, $3, $4, $5)) 
			limit 
				20000`,
			geom, columns, schema, prefix, layer,
		),
	}, nil
}

// Layer represents a database table
type Layer struct {
	Name  string
	Query string
}

// logger implementation for the PGX interface - this will wrap the Logurus log and map to it's
// levels.
type logger struct {
}

func (l *logger) Log(level pgx.LogLevel, msg string, data map[string]interface{}) {
	switch level {
	case pgx.LogLevelTrace:
	case pgx.LogLevelDebug:
		log.WithFields(data)
		log.Debug(msg)

	case pgx.LogLevelInfo:
		log.WithFields(data)
		log.Info(msg)

	case pgx.LogLevelWarn:
		log.WithFields(data)
		log.Warn(msg)

	case pgx.LogLevelError:
		log.WithFields(data)
		log.Error(msg)

	case pgx.LogLevelNone:
		log.WithFields(data)
		log.Info(msg)

	default:
		log.WithFields(data)
		log.Debug(msg)

	}

}
