@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"
for %%I in ("%SCRIPT_DIR%.") do set "SCENE_DIR=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"

call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%\scene.json" ^
  --script "%SCENE_DIR%\prism-absorbing.json" ^
  --script "%SCENE_DIR%\beauty.json" ^
  %*

exit /b %ERRORLEVEL%
