# Changelog

All notable changes to this project will be documented in this file.

## [0.2.2] - 2025-06-10

### Added
- Support for Tailwind CSS v4 integration
- Django REST Framework setup option
- VS Code integration with automatic project opening
- Multiple terminal support for development servers
- Enhanced project templates with modern UI components

### Improved
- Better error handling and validation messages
- Streamlined virtual environment setup with uv
- More detailed progress tracking during setup
- Enhanced template context handling
- Modern landing page with Tailwind CSS styling

### Fixed
- Virtual environment path issues on Windows
- Template context injection reliability
- Project name validation edge cases
- Terminal command execution on different platforms

## [0.2.1] - 2025-06-04

### Added
- Project name now appears in welcome page and documentation
- Improved template structure with proper context passing
- Enhanced API documentation page with better styling
- Centralized layout for both welcome and API documentation pages

### Fixed
- Fixed project name not displaying in templates
- Resolved URL namespace conflicts with REST framework
- Improved template context handling
- Fixed centering issues in template layouts

## [0.2.0] - 2025-06-02

### Added

-   Dynamic progress bar based on actual completed steps
-   Cross-platform build support for Windows, macOS, and Linux
-   Comprehensive README.md documentation with detailed usage instructions
-   Configuration file support for user defaults
-   Command-line argument support with flags (`-n`, `-v`, `--auto`, `--help`)
-   Non-interactive mode with `--auto` flag for automated project creation

### Improved

-   Progress bar now accurately reflects project creation progress
-   Enhanced user interface with better color schemes and styling
-   Optimized progress bar animation and immediate visual feedback
-   Better error handling and validation messages
-   Improved form styling with custom placeholder colors
-   Streamlined project creation flow without blank screens

### Fixed

-   Resolved compilation errors when building for different platforms
-   Fixed blank screen issue after project configuration selection
-   Corrected placeholder text color overlapping issues
-   Improved theme configuration for better visual consistency

### Technical

-   Updated build process with proper cross-compilation support
-   Enhanced Go module dependencies and version management
-   Improved code organization and documentation
-   Added comprehensive troubleshooting guide

## [0.1.0] - 2025-05-03

### Added

-   Git repository initialization option
-   Automatic `.gitignore` file creation for Django projects

-   Initial release of Django CLI tool
-   Project creation with customizable Django version
-   Virtual environment setup using `uv`
-   Django project initialization with basic structure
-   Optional templates and static files setup
    -   Global templates directory with base.html and index.html
    -   Static files structure (CSS and JS)
    -   Basic styling and JavaScript setup
-   Django app creation with automatic registration
    -   App-specific templates directory
    -   Integration with project settings
-   Development server launch option

### Features

-   Git initialization and .gitignore creation

-   Interactive CLI interface with step-by-step project setup
-   Progress tracking and status messages
-   Error handling and validation
-   Configurable project structure
-   Support for custom Django versions
