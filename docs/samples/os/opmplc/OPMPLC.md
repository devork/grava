# Open Map Local

This demo is of the [Open Map Local](https://www.ordnancesurvey.co.uk/business-and-government/products/os-open-map-local.html) dataset from the Ordnance Survey. 

# Running

+ Fetch the `SU` shapefile tile data from the Opendata link
+ Create a postgres/postgis database (see the [pg_setup.sql](../pg_setup.sql) file)
+ Unzip into a directory (e.g. `shp`)
+ Run the script `opmplc_shp_pg.sh` to import the data (DB details are based on the `pg_setup.sql` script)
+ Run the `gravad` deamon: `${PATH}/gravad --with-cors --config opmplc.config.json`
+ Serve the `index.html` file, e.g. using python - *note the path!*: `cd ${GRAVA_HOME}/docs/samples/os && python -m SimpleHTTPServer 8081`

# Style

The style is adapted from the Ordnance Survey style sheets - available at [Github](https://github.com/OrdnanceSurvey/OS-OpenMap-Local-stylesheets). Not all of the style is implemented but enough for a sample.