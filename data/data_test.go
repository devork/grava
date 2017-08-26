package data

import (
	"testing"

	"github.com/devork/geom"
	"github.com/devork/grava/geo"
	"github.com/stretchr/testify/require"
)

// https://github.com/mapbox/vector-tile-spec/tree/master/2.1#4355-example-polygon
func TestReadPolygon(t *testing.T) {
	polygon := &geom.Polygon{
		Hdr: geom.Hdr{Dim: geom.XY, Srid: 27700},
		Rings: []geom.LinearRing{{
			Coordinates: []geom.Coordinate{
				{3, 6},
				{8, 12},
				{20, 34},
				{3, 6},
			},
		}},
	}

	feature := readPolygon(polygon, &geo.BBox{}, 1.0, 1.0)
	require.NotNil(t, feature)

	t.Logf("%+v", feature.Geometry)
	require.Equal(t, 9, len(feature.Geometry))

	var expected = []uint32{9, 6, 12, 18, 10, 12, 24, 44, 15}
	for idx, cmd := range feature.Geometry {
		require.Equal(t, expected[idx], cmd)
	}
}
