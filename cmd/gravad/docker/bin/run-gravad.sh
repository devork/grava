#!/bin/sh

if [ "${GRAVAD_CONFIG}" == "" ]; then
    export GRAVAD_CONFIG=/etc/gravad/config.json
fi

echo "Running gravad:"
echo "  Configuration: ${GRAVAD_CONFIG}"

su -l -s /bin/sh -c "/usr/local/bin/gravad --config ${GRAVAD_CONFIG}"