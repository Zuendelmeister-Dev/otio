# Python 3.10+ and Docker Compose or PostgreSQL client tools are required.
# Example: ./scripts/historian.ps1 backup --compose docker-compose.yml --output backups/demo.dump
$ErrorActionPreference = 'Stop'

# Prefer the installed launcher and verify candidates. Windows may expose a
# python.exe Store alias on PATH even when no usable interpreter is installed.
foreach ($candidate in @('py', 'python', 'python3')) {
    $pythonCommand = Get-Command $candidate -ErrorAction SilentlyContinue
    if (-not $pythonCommand) { continue }
    $pythonOptions = @()
    if ($candidate -eq 'py') { $pythonOptions = @('-3') }
    try {
        & $pythonCommand.Source @pythonOptions -c 'import sys; sys.exit(0 if sys.version_info >= (3, 10) else 1)' 2>$null
        if ($LASTEXITCODE -ne 0) { continue }
    }
    catch { continue }
    & $pythonCommand.Source @pythonOptions (Join-Path $PSScriptRoot 'historian.py') @args
    exit $LASTEXITCODE
}
throw 'Install Python 3.10 or newer and add it to PATH.'
