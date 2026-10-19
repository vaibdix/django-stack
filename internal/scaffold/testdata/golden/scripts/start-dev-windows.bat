@echo off
setlocal
set "PROJECT_DIR=%~dp0"

where code >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    start "" code -n "%PROJECT_DIR%."
    echo VS Code opened in a new window.
    echo Allow automatic tasks when VS Code asks, or run "Workspace: Start Dev" from the Tasks menu.
    echo Watch the Django terminal inside VS Code.
) else (
    echo The code command is not on your PATH. Open this folder in VS Code yourself.
)
echo Open http://localhost:8000 once the server is ready.
