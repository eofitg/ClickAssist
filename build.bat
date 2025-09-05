@echo off
mkdir dist

echo Building for Windows...
go build -o dist\ClickAssist_win.exe main.go

echo Done. Check the dist folder.
pause
