@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"
for %%I in ("%SCRIPT_DIR%.\") do set "SCENE_DIR=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

if not defined RUN_STAMP (
  for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss"') do set "RUN_STAMP=%%I"
)
set "RAY_STUDIO_INTERMEDIATE_STAMP=%RUN_STAMP%"
set "RUN_TMP=%REPO_ROOT%\outputs\temp\geometry-benchmark-matrix-endless-%RUN_STAMP%"
set "CHECKPOINT_DIR=%REPO_ROOT%\outputs\geometry-benchmark-matrix-endless-%RUN_STAMP%"
if not exist "%RUN_TMP%" mkdir "%RUN_TMP%"
if not exist "%CHECKPOINT_DIR%" mkdir "%CHECKPOINT_DIR%"
set "TMP=%RUN_TMP%"
set "TEMP=%RUN_TMP%"

echo Geometry benchmark endless stamp: %RUN_STAMP%
echo Checkpoint dir: %CHECKPOINT_DIR%

call "%SCENE_DIR%\run.cmd" ^
  --endless ^
  --checkpoint-interval 50 ^
  --checkpoint-dir "%CHECKPOINT_DIR%" ^
  %*

exit /b %ERRORLEVEL%
