@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..\..\..") do set "REPO_ROOT=%%~fI"
for %%I in ("%SCRIPT_DIR%.") do set "SCENE_DIR=%%~fI"

if not defined npm_config_cache set "npm_config_cache=%REPO_ROOT%\.npm-cache"
set "VERIFY_DIR=%REPO_ROOT%\outputs\prism-spectrum-verify"
if not exist "%VERIFY_DIR%" mkdir "%VERIFY_DIR%"

echo Rendering lossless control prism...
call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%\scene.json" ^
  --script "%SCENE_DIR%\prism-control.json" ^
  --script "%SCENE_DIR%\diagnostic.json" ^
  --script "%SCENE_DIR%\verify-control.json" ^
  %*
if errorlevel 1 exit /b %ERRORLEVEL%

echo Rendering absorbing prism...
call npm --prefix "%REPO_ROOT%" run studio -- ^
  --script "%SCENE_DIR%\scene.json" ^
  --script "%SCENE_DIR%\prism-absorbing.json" ^
  --script "%SCENE_DIR%\diagnostic.json" ^
  --script "%SCENE_DIR%\verify-absorbing.json" ^
  %*
if errorlevel 1 exit /b %ERRORLEVEL%

echo Checking control dispersion...
call go -C "%REPO_ROOT%\engine" run ./cmd/spectral_film_probe ^
  --film "%VERIFY_DIR%\control.bin" ^
  --verify-prism ^
  --summary-only
if errorlevel 1 exit /b %ERRORLEVEL%

echo Checking absorbing dispersion and energy ratios...
call go -C "%REPO_ROOT%\engine" run ./cmd/spectral_film_probe ^
  --film "%VERIFY_DIR%\absorbing.bin" ^
  --reference "%VERIFY_DIR%\control.bin" ^
  --verify-prism ^
  --verify-absorption ^
  --summary-only

exit /b %ERRORLEVEL%
