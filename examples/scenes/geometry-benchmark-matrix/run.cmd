@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

call npm --prefix "%REPO_ROOT%" run ray -- ^
  --script "%SCRIPT_DIR%room.json" ^
  --script "%SCRIPT_DIR%main.json" ^
  --script "%SCRIPT_DIR%materials.json" ^
  --script "%SCRIPT_DIR%geo-r01.json" ^
  --script "%SCRIPT_DIR%geo-r02.json" ^
  --script "%SCRIPT_DIR%geo-r03.json" ^
  --script "%SCRIPT_DIR%geo-r04.json" ^
  --script "%SCRIPT_DIR%geo-r05.json" ^
  --script "%SCRIPT_DIR%geo-r06.json" ^
  %*

exit /b %ERRORLEVEL%
