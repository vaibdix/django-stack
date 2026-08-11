@echo off
echo ========================================
echo   Django Development Environment
echo ========================================
echo.
echo Opening VS Code...
echo.

echo [1/1] Opening VS Code...
start "" "/usr/local/bin/code" -n "/home/dev/my-shop"

echo.
echo ========================================
echo   VS Code opened
echo ========================================
echo.
echo   The workspace auto-runs the "Workspace: Start Dev" task.
echo   Use the integrated terminals inside VS Code for Django and Tailwind.
echo.
pause
