@echo off
pushd src
go build .
popd
xcopy src\Raku.exe release\windows\ /Y
