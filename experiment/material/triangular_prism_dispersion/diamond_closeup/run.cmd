@echo off
setlocal

set "SCENE_DIR=%~dp0"
for %%I in ("%SCENE_DIR%..\..\..\..") do set "REPO_ROOT=%%~fI"

if not exist "%SCENE_DIR%results" mkdir "%SCENE_DIR%results"
if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%main.json" ^
  --script "%SCENE_DIR%set.json" ^
  --script "%SCENE_DIR%..\moissanite.json" ^
  --script "%SCENE_DIR%gem.json" ^
  --script "%SCENE_DIR%lights.json" ^
  --output-image "%SCENE_DIR%results\diamond-closeup.png" ^
  --output-film "%SCENE_DIR%results\diamond-closeup.bin" ^
  %*

exit /b %ERRORLEVEL%
