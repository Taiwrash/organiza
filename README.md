# Organiza

Organiza is a quiet, minimalist macOS application designed to keep your Desktop clear by automatically sorting files into dedicated folders based on their extensions.

Designed with **Dieter Rams' Principles** in mind: It is as little design as possible, unobtrusive, and honest.

## Features
- **Real-time Monitoring**: Automatically detects new files on your Desktop and moves them instantly.
- **Smart Sorting**: Organizes files by extension (e.g., `.pdf` → `pdf/`, `.png` → `png/`).
- **Data Integrity**: Automatically appends suffixes to duplicate filenames to prevent accidental overwrites.
- **Download Safety**: Intelligently waits for active downloads and large file writes to complete before moving.
- **Privacy First**: Runs entirely locally on your machine. No accounts, no cloud, no tracking.

## Getting Started

### Prerequisites
- **Go** 1.26+
- **Node.js** 18+
- **Wails CLI** v2.12.0+

### Development
To run the application in development mode with hot-reloading:
```bash
wails dev
```

### Build
To create a production-ready binary:
```bash
wails build
```
The executable will be located in `build/bin/`.

## Project Structure
- `/backend`: Go logic and file system watcher.
- `/frontend`: React/Vite minimalist UI.
- `/website`: Standalone marketing website.

## Security & Maintenance
- **Vulnerability Patches**: Backend file operations patched for race conditions and silent overwrites.
- **Modern Dependencies**: Frontend stack upgraded to Vite 5 and Rollup 4.
- **Toolchain**: Optimized for Go 1.26.2.

---
© 2026 Rasheed Mudasiru
