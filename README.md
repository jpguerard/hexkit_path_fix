# hexkit_path_fix

[![Go Report Card](https://goreportcard.com/badge/github.com/jpguerard/hexkit_path_fix)](https://goreportcard.com/report/github.com/jpguerard/hexkit_path_fix)

Fix an [HexKit](http://www.hex-kit.com/) map file after some hex have
been moved

*Tested on Windows and Linux.*

hexkit_path_fix will update an HexKit map to work with the current
tiles positions.

It can be used to fix a map before using it on another computer with
a different tile organization.

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
    REM On Windows
    hexkit_path_fix.exe "C:\Hex Kit-win32-x64" Octarine.map > NewOctarine.map

    # On Linux
    hexkit_path_fix "~/RPG_Mapping/Hex Kit-linux-x64" Octarine.map > NewOctarine.map

```

Always check the new map with HexKit before erasing the old map.

Limitations:

 - Map files include a copy of the generator used to build them. This is used to
   add new rows or columns. Single tiles and folder names used in generators are
   not currently converted.

