# hexkit_path_fix
Fix an [HexKit](http://www.hex-kit.com/) map file after some hex have
been moved

*Tested on Windows and Linux.*

Usage :

 - Copy the executable in the HexKit folder (where your hexagon
   collections are stored).

 - From a command window, run:

```
    REM On Windows
    hexkit_path_fix.exe HexKitPath  mapPath > newMap

    # On Linux
    ./hexkit_path_fix HexKitPath mapPath > newMap
```

For example:

```
    # On Windows
    hexkit_path_fix.exe "C:\Hex Kit-win32-x64" Octarine.map > NewOctarine.map

    # On Linux
    hexkit_path_fix "~/RPG_Mapping/Hex Kit-linux-x64" Octarine.map > NewOctarine.map

```

Always check the new map with HexKit before erasing the old map.

Limitations:

 - Generators are not currently converted.

