#!/bin/dash

build() {
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags="-s -w"
}

make_tar() {
  echo " * $3"
  build "$1" "$2"
  file hexkit_path_fix
  tar cvjf "release/hexkit_path_fix_$3.tar.bz2" hexkit_path_fix LICENSE README.md
}

make_zip() {
  echo " * $3"
  build "$1" "$2"
  file hexkit_path_fix.exe
  zip "release/hexkit_path_fix_$3.zip" hexkit_path_fix.exe LICENSE README.md
}

rm -f release/*
rmdir release
mkdir -p release
go mod tidy
make_zip windows 386 win32
make_zip windows amd64 win64
make_tar darwin amd64 darwin64
make_tar linux 386 linux32
make_tar linux amd64 linux64
rm -f hexkit_path_fix.exe
