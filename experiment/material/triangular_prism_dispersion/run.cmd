@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"
for %%I in ("%SCRIPT_DIR%.") do set "SCENE_DIR=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

if not defined RUN_STAMP (
  for /f %%I in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss"') do set "RUN_STAMP=%%I"
)
set "CHECKPOINT_DIR=%SCENE_DIR%\results\endless-%RUN_STAMP%"
if not exist "%CHECKPOINT_DIR%" mkdir "%CHECKPOINT_DIR%"

echo Triangular prism dispersion endless stamp: %RUN_STAMP%
echo Checkpoint dir: %CHECKPOINT_DIR%

call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%\main.json" ^
  --script "%SCENE_DIR%\room.json" ^
  --script "%SCENE_DIR%\prism.json" ^
  --script "%SCENE_DIR%\prism-scene.json" ^
  --script "%SCENE_DIR%\moissanite.json" ^
  --script "%SCENE_DIR%\moissanite-scene.json" ^
  --endless ^
  --checkpoint-interval 1 ^
  --checkpoint-dir "%CHECKPOINT_DIR%" ^
  %*

exit /b %ERRORLEVEL%
