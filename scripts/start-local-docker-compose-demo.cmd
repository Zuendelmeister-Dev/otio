@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-local-docker-compose-demo.ps1" %*
exit /b %ERRORLEVEL%
