package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
)

type FileManager struct {
	app          *tview.Application
	list         *tview.List
	fileDetails  *tview.TextView
	gitDetails   *tview.TextView
	terminalView *tview.TextView
	ptyFile      *os.File
	path         string
	items        []string
	oldState     *term.State
	flex         *tview.Flex
	focusOnTerm  bool
	vimRunning   bool
}

func NewFileManager(path string) *FileManager {
	fm := &FileManager{
		app:          tview.NewApplication(),
		list:         tview.NewList(),
		fileDetails:  tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		gitDetails:   tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		terminalView: tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		path:         path,
		flex:         tview.NewFlex(),
	}

	// Configure terminal view
	fm.terminalView.SetBorder(true).SetTitle("Terminal")

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
		info, err := os.Stat(filepath.Join(fm.path, item))
		if err != nil {
			continue
		}

		var color tcell.Color
		var symbol string

		switch mode := info.Mode(); {
		case mode.IsDir():
			color = tcell.ColorBlue
			symbol = "📁"
		case mode&os.ModeSymlink != 0:
			color = tcell.ColorFuchsia
			symbol = "🔗"
		case mode&os.ModeNamedPipe != 0:
			color = tcell.ColorYellow
			symbol = "📜"
		case mode&os.ModeSocket != 0:
			color = tcell.ColorGreen
			symbol = "🔌"
		case mode&os.ModeDevice != 0:
			color = tcell.ColorRed
			symbol = "🔧"
		default:
			color = tcell.ColorWhite
			symbol = "📄"
		}

		fm.list.AddItem(fmt.Sprintf("%s %s", symbol, item), "", 0, func(name string) func() {
			return func() { fm.navigate(name) }
		}(item)).SetMainTextColor(color)
	}

	fm.updateDetails()
	fm.updateGitDetails()
}

func isGitRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	stat, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return stat.IsDir() || (stat.Mode()&os.ModeSymlink != 0)
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
	if fm.vimRunning {
		return
	}
	fm.vimRunning = true

	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	fm.oldState = oldState

	fm.app.Suspend(func() {
		cmd := exec.Command("vim", filepath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()

		if fm.oldState != nil {
			_ = term.Restore(int(os.Stdin.Fd()), fm.oldState)
		}

		fm.vimRunning = false
	})

	fm.loadItems()
}

func (fm *FileManager) updateDetails() {
	idx := fm.list.GetCurrentItem()
	if idx < 0 || idx >= len(fm.items) {
		fm.fileDetails.SetText("")
		return
	}

	item := fm.items[idx]
	full := filepath.Join(fm.path, item)
	info, err := os.Stat(full)
	if err != nil {
		fm.fileDetails.SetText(fmt.Sprintf("Error: %v", err))
		return
	}

	fType := "File"
	if info.IsDir() {
		fType = "Directory"
	} else if info.Mode()&os.ModeSymlink != 0 {
		fType = "Symlink"
	}

	perms := info.Mode().String()
	stat := info.Sys().(*syscall.Stat_t)
	owner := stat.Uid
	group := stat.Gid
	mod := info.ModTime().Format(time.RFC1123)
	size := info.Size()

	details := fmt.Sprintf(
		"[yellow]Name:[white] %s\n[yellow]Type:[white] %s\n[yellow]Size:[white] %d bytes\n[yellow]Permissions:[white] %s\n[yellow]Owner:[white] %d\n[yellow]Group:[white] %d\n[yellow]Modified:[white] %s",
		info.Name(), fType, size, perms, owner, group, mod,
	)
	fm.fileDetails.SetText(details)
}

func (fm *FileManager) deleteSelectedItem() {
	idx := fm.list.GetCurrentItem()
	if idx < 0 || idx >= len(fm.items) {
		return
	}
	full := filepath.Join(fm.path, fm.items[idx])
	_ = os.RemoveAll(full)
	fm.loadItems()
}

func (fm *FileManager) updateGitDetails() {
	if !isGitRepo(fm.path) {
		fm.gitDetails.SetText("")
		return
	}
	fm.gitDetails.SetText(fm.gitLog())
}

func (fm *FileManager) gitLog() string {
	cmd := exec.Command("git", "-C", fm.path, "log", "--oneline", "-10")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	var sb strings.Builder
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			sb.WriteString(fmt.Sprintf("[yellow]%s [white]%s\n", parts[0], parts[1]))
		}
	}
	return sb.String()
}

func (fm *FileManager) startTerminal() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.Command(shell)
	f, err := pty.Start(cmd)
	if err != nil {
		fm.terminalView.SetText(fmt.Sprintf("Error starting shell: %v", err))
		return
	}
	fm.ptyFile = f

	go func() {
		buf := make([]byte, 256)
		for {
			n, err := f.Read(buf)
			if err != nil {
				return
			}
			fm.app.QueueUpdateDraw(func() {
				fm.terminalView.Write(buf[:n])
			})
		}
	}()
}

func (fm *FileManager) run() {
    top := tview.NewFlex().SetDirection(tview.FlexColumn).
        AddItem(fm.list, 0, 1, true).
        AddItem(tview.NewBox().SetBorder(false), 1, 0, false).
        AddItem(fm.fileDetails, 0, 2, false).
        AddItem(tview.NewBox().SetBorder(false), 1, 0, false).
        AddItem(fm.gitDetails, 0, 2, false)

    fm.flex.SetDirection(tview.FlexRow).
        AddItem(top, 0, 1, true).
        AddItem(fm.terminalView, 0, 1, false)

    fm.startTerminal()

    // Update file details on selection change
    fm.list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
        fm.updateDetails()
    })

    // Navigate into directories or open files
    fm.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
        if index >= 0 && index < len(fm.items) {
            fm.navigate(fm.items[index])
        }
    })

    // Global key handling
    fm.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {
        case tcell.KeyTab:
            fm.focusOnTerm = !fm.focusOnTerm
            if fm.focusOnTerm {
                fm.app.SetFocus(fm.terminalView)
            } else {
                fm.app.SetFocus(fm.list)
            }
        case tcell.KeyDelete:
            fm.deleteSelectedItem()
        case tcell.KeyEsc:
            fm.app.Stop()
        }
        return event
    })

    if err := fm.app.SetRoot(fm.flex, true).Run(); err != nil {
        panic(err)
    }
}

func main() {
	fm := NewFileManager(".")
	fm.run()
}

