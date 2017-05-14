# hexkit_path_fix
Fix an HexKit map file after some hex have been moved

*This software is alpha quality. Use with care.*
*This software has only been tested on Linux.*

Usage :

 - Copy the executable in the HexKit folder (where your hexagon
   collections are stored).

 - From a command window, run:

    hexkit_path_fix   hexKitPath mapPath > newMap (for Windows)
    ./hexkit_path_fix hexKitPath mapPath > newMap (for Linux)_

   For example:
    ./hexkit_path_fix ~/RPG__Mapping/HexKit ~/MyMaps/Test.map > ~/NewMap.map

   The program will run a search for all PNG files under the hexkit directory
   and use this information to locate the new tile position.

Do not use the same name for the new map, you do not
want to lose it should there be an problem.

Moving tiles to a new collection is currently not supported.
