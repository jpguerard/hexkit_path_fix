# hexkit_path_fix
Fix an [HexKit](http://www.hex-kit.com/) map file after some hex have
been moved

*This software is alpha quality. Use with care.*
*Tested on Windows and Linux.*

Usage :

 - Copy the executable in the HexKit folder (where your hexagon
   collections are stored).

 - From a command window, run:

```
    REM On Windows
    hexkit_path_fix   CollectionPath... mapPath > newMap

    # On Linux
    ./hexkit_path_fix CollectionPath... mapPath > newMap
```

For example:

```
    # The HexKit directory contains all collections
    ./hexkit_path_fix RPG_Mapping/HexKit MyMaps/Test.map > ~/NewMap.map

    # List individual collections
    ./hexkit_path_fix "RPG_Mapping/HexKit/HK-Fantasy" "RPG_Mapping/HexKit/HK-Traveling Through Dangerous Scenery" ~/MyMaps/Test.map > ~/NewMap.map
```

The program will run a search for all PNG files under the hexkit directory
and use this information to locate the new tile position.

Do not use the same name for the new map, you do not
want to lose it should there be an problem.

Limitations:

 - Moving tiles to a new collection is currently not supported.
 - Generators are not currently converted.

