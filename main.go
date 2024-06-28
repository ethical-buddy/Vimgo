// package main

// import (
// 	"os"
// 	"os/exec"
// 	"path/filepath"
// 	"sort"

// 	"github.com/gdamore/tcell/v2"
// 	"github.com/rivo/tview"
// 	"golang.org/x/term"
// )

// type FileManager struct {
// 	app      *tview.Application
// 	list     *tview.List
// 	path     string
// 	items    []string
// 	oldState *term.State // To store original terminal state
// }

// func NewFileManager(path string) *FileManager {
// 	fm := &FileManager{
// 		app:  tview.NewApplication(),
// 		list: tview.NewList(),
// 		path: path,
// 	}

// 	fm.loadItems()
// 	return fm
// }

// func (fm *FileManager) loadItems() {
// 	fm.list.Clear()
// 	entries, err := os.ReadDir(fm.path)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var dirs, files []string
// 	for _, entry := range entries {
// 		if entry.IsDir() {
// 			dirs = append(dirs, entry.Name())
// 		} else {
// 			files = append(files, entry.Name())
// 		}
// 	}

// 	sort.Strings(dirs)
// 	sort.Strings(files)

// 	fm.items = append(dirs, files...)
// 	for _, item := range fm.items {
// 		fm.list.AddItem(item, "", 0, nil)
// 	}
// }

// func (fm *FileManager) navigate(item string) {
// 	newPath := filepath.Join(fm.path, item)
// 	info, err := os.Stat(newPath)
// 	if err != nil {
// 		return
// 	}

// 	if info.IsDir() {
// 		fm.path = newPath
// 		fm.loadItems()
// 	} else {
// 		fm.openInVim(newPath)
// 	}
// }

// func (fm *FileManager) openInVim(filepath string) {
// 	// Save current terminal state
// 	oldState, err := term.GetState(int(os.Stdin.Fd()))
// 	if err != nil {
// 		panic(err)
// 	}
// 	fm.oldState = oldState

// 	// Clear the screen to make space for Vim
// 	exec.Command("clear").Run()

// 	// Open the file in Vim
// 	cmd := exec.Command("vim", filepath)
// 	cmd.Stdin = os.Stdin
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	// Start Vim
// 	if err := cmd.Start(); err != nil {
// 		panic(err)
// 	}

// 	// Wait for Vim to finish
// 	if err := cmd.Wait(); err != nil {
// 		panic(err)
// 	}

// 	// Restore terminal state after Vim exits
// 	if fm.oldState != nil {
// 		if err := term.Restore(int(os.Stdin.Fd()), fm.oldState); err != nil {
// 			panic(err)
// 		}
// 	}

// 	// Re-run the file manager application after closing Vim
// 	fm.app.Draw()
// }

// func (fm *FileManager) run() {
// 	fm.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
// 		fm.navigate(mainText)
// 	})

// 	fm.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
// 		switch event.Key() {
// 		case tcell.KeyBackspace, tcell.KeyBackspace2:
// 			fm.path = filepath.Dir(fm.path)
// 			fm.loadItems()
// 			return nil
// 		case tcell.KeyRune:
// 			switch event.Rune() {
// 			case 'q':
// 				fm.app.Stop()
// 				return nil
// 			}
// 		}
// 		return event
// 	})

// 	fm.app.SetRoot(fm.list, true)

// 	if err := fm.app.Run(); err != nil {
// 		panic(err)
// 	}
// }

// func main() {
// 	fm := NewFileManager(".")
// 	fm.run()
// }

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
)

type FileManager struct {
	app      *tview.Application
	list     *tview.List
	path     string
	items    []string
	oldState *term.State // To store original terminal state
}

func NewFileManager(path string) *FileManager {
	fm := &FileManager{
		app:  tview.NewApplication(),
		list: tview.NewList(),
		path: path,
	}

	fm.loadItems()
	return fm
}

func (fm *FileManager) loadItems() {
	fm.list.Clear()
	entries, err := os.ReadDir(fm.path)
	if err != nil {
		panic(err)
	}

	var dirs, files []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(dirs)
	sort.Strings(files)

	fm.items = append(dirs, files...)
	for _, item := range fm.items {
		fm.list.AddItem(item, "", 0, nil)
	}
}

func (fm *FileManager) navigate(item string) {
	newPath := filepath.Join(fm.path, item)
	info, err := os.Stat(newPath)
	if err != nil {
		return
	}

	if info.IsDir() {
		fm.path = newPath
		fm.loadItems()
	} else {
		fm.openInVim(newPath)
	}
}

func (fm *FileManager) openInVim(filepath string) {
	// Save current terminal state
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	fm.oldState = oldState

	// Exit the tview application before opening Vim
	fm.app.Suspend(func() {
		// Clear the screen to make space for Vim
		exec.Command("clear").Run()

		// Open the file in Vim
		cmd := exec.Command("vim", filepath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Start Vim
		if err := cmd.Run(); err != nil {
			panic(err)
		}

		// Restore terminal state after Vim exits
		if fm.oldState != nil {
			if err := term.Restore(int(os.Stdin.Fd()), fm.oldState); err != nil {
				panic(err)
			}
		}
	})
}

func (fm *FileManager) run() {
	fm.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		fm.navigate(mainText)
	})

	fm.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			fm.path = filepath.Dir(fm.path)
			fm.loadItems()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				fm.app.Stop()
				return nil
			}
		}
		return event
	})

	fm.app.SetRoot(fm.list, true)

	if err := fm.app.Run(); err != nil {
		panic(err)
	}
}

func main() {
	fm := NewFileManager(".")
	fm.run()
}
