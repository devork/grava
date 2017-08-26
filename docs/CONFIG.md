# Configuration

The service works by serving tables from a database as individual layers. These layers map 1-to-1 with the layer definitions in the styling JSON
of the Mapbox style.

## Core configuration

| Element       | Description                                                       |
|:--------------|:------------------------------------------------------------------|
| `fontsDir`    | Root directory to serve fonts from                                |
| `postgres`    | Postgres URI schema for connection details                        |
| `schema`      | The schema to fetch layers from                                   |
| `sources`     | List of source definitions                                        |


### Fonts

The font directory contains the PBF derived fonts (see [FONTS.md](FONTS.md)). 

The directory is then composed of `font stack` directories , e.g. `OpenSansSemiBold`. These correspond to the names used in the `text-font` attributes in the style definition.

## Sources

A source corresponds to a grouping of data within a tile. In the example below, one source is defined (called `opmplc`) - tiles served from this source, will contain deta taken from the layers:

    woodland
    tidalwater
    surfacewater_area
    glasshouse
    building
    importantbuilding
    roadtunnel
    road
    roundabout
    railwaytrack
    railwaystation
    namedplace

These sources map to the JSON configuation of a style.

Source endpoints are defined as `http://host:port/{source}`

## Sample Configuration

The following is taken from the Open Map Place demo:

    {
        "fontsDir": "../../fonts",
        "postgres": "postgresql://mvt_user:9Ep7XgAMDZnYW6V@127.0.0.1/mvt?sslmode=disable",
        "schema": "grava",
        "sources": [
            {
                "prefix": "opmplc_",
                "name": "opmplc",
                "layers":[
                    "woodland",
                    "tidalwater",
                    "surfacewater_area",
                    "glasshouse",
                    "building",
                    "importantbuilding",
                    "roadtunnel",
                    "road",
                    "roundabout",
                    "railwaytrack",
                    "railwaystation",
                    "namedplace"
                ]
            }
        ]
    }