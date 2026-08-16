@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..\..") do set "REPO_ROOT=%%~fI"
for %%I in ("%SCRIPT_DIR%..\") do set "SCENE_DIR=%%~fI"
for %%I in ("%SCRIPT_DIR%.\") do set "EXAMPLE_DIR=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

if not defined RUN_STAMP (
  for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss"') do set "RUN_STAMP=%%I"
)
set "RAY_STUDIO_INTERMEDIATE_STAMP=%RUN_STAMP%"
set "RUN_TMP=%REPO_ROOT%\outputs\temp\geometry-benchmark-example-%RUN_STAMP%"
if not exist "%RUN_TMP%" mkdir "%RUN_TMP%"
set "TMP=%RUN_TMP%"
set "TEMP=%RUN_TMP%"
set "OUTPUT_IMAGE=%REPO_ROOT%\outputs\studio-geometry-benchmark-example-%RUN_STAMP%.png"
set "OUTPUT_FILM=%REPO_ROOT%\outputs\studio-geometry-benchmark-example-%RUN_STAMP%.bin"

echo Geometry benchmark example stamp: %RUN_STAMP%
echo Output image: %OUTPUT_IMAGE%
echo Output film: %OUTPUT_FILM%

call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%\room.json" ^
  --script "%SCENE_DIR%\main.json" ^
  --script "%SCENE_DIR%\materials.json" ^
  --script "%EXAMPLE_DIR%\geo_example.json" ^
  --output-image "%OUTPUT_IMAGE%" ^
  --output-film "%OUTPUT_FILM%" ^
  %*

exit /b %ERRORLEVEL%
