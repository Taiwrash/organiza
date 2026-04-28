package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (b *App) ListDirectories() []string {
	var listDir []string
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting user home directory: %v\n", err)
		return listDir
	}

	files, err := os.ReadDir(home)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return listDir
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			listDir = append(listDir, file.Name())
		}
	}

	return listDir
}

// SelectDirectory opens a native OS directory picker
func (b *App) SelectDirectory() string {
	selection, err := runtime.OpenDirectoryDialog(b.ctx, runtime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
	if err != nil {
		log.Println("Error selecting directory:", err)
		return ""
	}
	return selection
}

func (b *App) Watcher(dest string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}
	src := filepath.Join(home, "Desktop")
	dst := filepath.Join(home, dest)

	// If the user selected a path through the picker, it might be absolute.
	// If it's a relative path, join it with home.
	if filepath.IsAbs(dest) {
		dst = dest
	}

	movedCount := 0
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the destination directory and all its contents to prevent recursive loops
		if path == dst || strings.HasPrefix(path, dst+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			moved, err := b.organizeFiles(path, src, dst, false)
			if moved {
				movedCount++
			}
			if err != nil {
				// Don't kill the whole process for one bad file, just log it
				log.Println("Error organizing file during walk:", err)
			}
		}
		return nil
	})
	
	if err != nil {
		log.Println("Error walking directory:", err)
		return fmt.Errorf("cannot read Desktop (check macOS Privacy permissions): %w", err)
	}

	if movedCount > 0 {
		b.notifySystem("Organiza", fmt.Sprintf("Sorted %d existing files from Desktop.", movedCount))
	}

	// Watch for new files in the source directory
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Error creating watcher:", err)
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	// Set up cancellation context
	watchCtx, cancel := context.WithCancel(context.Background())
	b.cancelFunc = cancel

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-watchCtx.Done():
				// Stop signal received
				done <- true
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					// Skip if the new file is inside the destination directory
					if event.Name == dst || strings.HasPrefix(event.Name, dst+string(os.PathSeparator)) {
						continue
					}
					log.Println("New file created:", event.Name)
					_, err := b.organizeFiles(event.Name, src, dst, true)
					if err != nil {
						log.Println("Error organizing file:", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(src)
	if err != nil {
		log.Println("Error adding watcher:", err)
		return fmt.Errorf("watching Desktop (check macOS Privacy permissions): %w", err)
	}
	<-done
	return nil
}

// StopWatching allows the user to manually stop the background watcher
func (b *App) StopWatching() {
	if b.cancelFunc != nil {
		b.cancelFunc()
		b.cancelFunc = nil
		log.Println("Stopped watching.")
	}
}

func (b *App) organizeFiles(filePath string, sourceDir string, destinationDir string, shouldNotify bool) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	// If the path is a directory or a symlink, don't try to move it
	if fileInfo.IsDir() || fileInfo.Mode()&os.ModeSymlink != 0 {
		return false, nil
	}

	fileExtension := filepath.Ext(filePath)
	lowerExt := strings.ToLower(fileExtension)
	if lowerExt == ".crdownload" || lowerExt == ".part" || lowerExt == ".tmp" || lowerExt == ".download" {
		return false, nil // Ignore temporary download files
	}

	var targetDir string
	if len(fileExtension) < 2 {
		targetDir = filepath.Join(destinationDir, "Others")
	} else {
		targetDir = filepath.Join(destinationDir, b.getCategory(lowerExt))
	}

	// Create the target directory if it doesn't exist
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.MkdirAll(targetDir, os.ModePerm)
		if err != nil {
			return false, err
		}
	}

	// Move the file directly to the category directory
	baseName := filepath.Base(filePath)
	newPath := filepath.Join(targetDir, baseName)

	// Prevent overwriting existing files
	counter := 1
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			break
		}
		// File exists, append counter
		nameWithoutExt := strings.TrimSuffix(baseName, fileExtension)
		newPath = filepath.Join(targetDir, fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, fileExtension))
		counter++
	}

	err = b.moveFileWithRetry(filePath, newPath)
	if err != nil {
		return false, err
	}

	log.Println("File moved to:", newPath)
	
	if shouldNotify {
		// Notify the user natively via macOS so they know background magic just happened
		b.notifySystem("Organiza", fmt.Sprintf("Moved %s to %s", baseName, b.getCategory(lowerExt)))
	}

	return true, nil
}

// notifySystem uses macOS osascript to trigger a native desktop notification
func (b *App) notifySystem(title, message string) {
	// Escape quotes just in case the filename contains them
	safeMessage := strings.ReplaceAll(message, `"`, `\"`)
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, safeMessage, title)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Start() // Run asynchronously so it doesn't block the file sorting
}

// getCategory maps a file extension to a human-readable folder name
func (b *App) getCategory(ext string) string {
	switch ext {
	case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".pages", ".odt", ".epub", ".mobi":
		return "Documents"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".heic", ".tiff", ".bmp":
		return "Images"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".wmv", ".flv":
		return "Videos"
	case ".mp3", ".wav", ".m4a", ".flac", ".aac", ".ogg":
		return "Audio"
	case ".csv", ".xls", ".xlsx", ".numbers":
		return "Spreadsheets"
	case ".key", ".ppt", ".pptx":
		return "Presentations"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".pkg":
		return "Archives"
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".py", ".html", ".css", ".json", ".sh", ".yaml", ".yml", ".md":
		return "Code"
	case ".dmg", ".app", ".exe":
		return "Apps & Installers"
	default:
		return "Others"
	}
}

func (b *App) moveFileWithRetry(src, dst string) error {
	// Fast path: if the file hasn't been modified in the last 2 seconds, 
	// it's highly likely to be completely written. Move it instantly.
	if info, err := os.Stat(src); err == nil {
		if time.Since(info.ModTime()) > 2*time.Second {
			err = os.Rename(src, dst)
			if err == nil {
				return nil
			}
			// If rename fails (e.g. file lock), fall through to retry loop
		}
	}

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		// Get initial size
		info1, err := os.Stat(src)
		if err != nil {
			return err
		}

		time.Sleep(500 * time.Millisecond)

		// Get size after delay
		info2, err := os.Stat(src)
		if err != nil {
			return err
		}

		// If size is changing, it's still being written to
		if info1.Size() != info2.Size() {
			continue
		}

		err = os.Rename(src, dst)
		if err == nil {
			return nil
		}
		// Wait before retrying in case it's temporarily locked
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("failed to move file after retries")
}
