@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

call npm --prefix "%REPO_ROOT%" run ray -- ^
  --script "%SCRIPT_DIR%room.json" ^
  --script "%SCRIPT_DIR%main.json" ^
  --script "%SCRIPT_DIR%materials.json" ^
  --script "%SCRIPT_DIR%geo_example.json" ^
  --output-image "..\outputs\geometry-benchmark-example.png" ^
  --output-film "..\outputs\geometry-benchmark-example.bin" ^
  %*

exit /b %ERRORLEVEL%
