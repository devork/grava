# Font Setup

Use the mapbox `node-fontnik`. 

To install (example below is for OpenSans Semibold which is used in the OS demo):

    brew install boost --c++11 freetype protobuf --c++11
    git clone https://github.com/mapbox/node-fontnik.git    
    cd node-fontnik
    npm install --build-from-source
    cd bin
    <<fetch OpenSans-Semibold.ttf && unzip>> 
    build-glyphs OpenSans-Semibold.ttf glyphs/semibold/
