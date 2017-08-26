#!/bin/bash

# This script will import the 'SU' shapefiles for the OS OpenMap Local dataset. Available from the Ordancne Survey Opendata website:
#
# https://www.ordnancesurvey.co.uk/business-and-government/products/os-open-map-local.html
#
# When run, this script will import shapefiles from the shp directory. It is designed to overwrite the existing tables. Use this script to 
# design your own import scripts.
# 

for i in `ls shp/*shp`
do
 	NAME=$(echo $i | perl -ne '/[A-Z][A-Z]_(.+)\.shp/; print lc($1)')
    OPTS="-lco OVERWRITE=YES -lco GEOMETRY_NAME=geometry -lco SPATIAL_INDEX=YES -lco LAUNDER=YES -lco SRID=3857 -append -gt 65536 -t_srs EPSG:3857 --config PG_USE_COPY YES -nlt GEOMETRY"

	if [ $NAME = 'functionalsite' ]
	then
		OPTS="$OPTS -nlt MULTIPOLYGON"
	fi
	echo "-------------------------------------------------------------------------------------------------------------------------"
	echo "Processing SHP: $NAME"
	echo "Command: ogr2ogr -f PostgreSQL PG:'dbname=mvt host=127.0.0.1 user=mvt_admin active_schema=grava password=6Ug8e352942kQcz' -nln opmplc_${NAME} $OPTS $i"
	ogr2ogr -f PostgreSQL PG:'dbname=mvt host=127.0.0.1 user=mvt_admin active_schema=grava password=6Ug8e352942kQcz' -nln opmplc_${NAME} $OPTS $i
done