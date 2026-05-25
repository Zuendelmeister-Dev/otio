@echo off
setlocal
powershell -ExecutionPolicy Bypass -File "%~dp0preflight.ps1" %*
endlocal
