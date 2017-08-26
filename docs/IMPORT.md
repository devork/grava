# OGR

```
ogr2ogr -f SQLite -dsco SPATIALITE=YES -lco SPATIAL_INDEX=YES -nln osdata -lco LAUNDER=YES -lco SRID=27700 -sql "select OGR_GEOMETRY, concat('{\"id\":\"',ID,'\", \"featcode\":',cast(FEATCODE as character(32)),'}') as PROPS from HP_Building" buildings.sqlite HP_Building.shp
```
