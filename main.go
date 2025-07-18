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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
)

type FileManager struct {
	app          *tview.Application
	list         *tview.List
	fileDetails  *tview.TextView
	gitDetails   *tview.TextView
	commands     *tview.TextView
	commandInput *tview.InputField // Input field for command mode
	path         string
	items        []string
	oldState     *term.State // To store original terminal state
	flex         *tview.Flex
	vimRunning   bool
	commandMode  bool // Flag to track if command mode is active
}

func NewFileManager(path string) *FileManager {
	fm := &FileManager{
		app:          tview.NewApplication(),
		list:         tview.NewList(),
		fileDetails:  tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		gitDetails:   tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft),
		commands:     tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true),
		commandInput: tview.NewInputField().SetLabel("Command: ").SetFieldWidth(0).SetFieldBackgroundColor(tcell.ColorBlack).SetLabelColor(tcell.ColorYellow),
		path:         path,
		flex:         tview.NewFlex(),
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
		case (mode & os.ModeSymlink) != 0:
			color = tcell.ColorFuchsia
			symbol = "🔗"
		case (mode & os.ModeNamedPipe) != 0:
			color = tcell.ColorYellow
			symbol = "📜"
		case (mode & os.ModeSocket) != 0:
			color = tcell.ColorGreen
			symbol = "🔌"
		case (mode & os.ModeDevice) != 0:
			color = tcell.ColorRed
			symbol = "🔧"
		default:
			color = tcell.ColorWhite
			symbol = "📄"
		}

		fm.list.AddItem(fmt.Sprintf("%s %s", symbol, item), "", 0, func() {
			fm.navigate(item)
		}).SetMainTextColor(color)
	}

	// Update both file and git details simultaneously
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
	} else if item == ".." {
		fm.path = filepath.Dir(fm.path)
	} else {
		fm.openInVim(newPath)
		return
	}

	fm.loadItems()
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
		cmd := exec.Command("nvim", filepath)
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
	selectedIndex := fm.list.GetCurrentItem()
	if selectedIndex < 0 || selectedIndex >= len(fm.items) {
		fm.fileDetails.SetText("")
		return
	}

	selectedItem := fm.items[selectedIndex]
	fullPath := filepath.Join(fm.path, selectedItem)

	info, err := os.Stat(fullPath)
	if err != nil {
		fm.fileDetails.SetText(fmt.Sprintf("Error retrieving details: %v", err))
		return
	}

	fileType := "File"
	if info.IsDir() {
		fileType = "Directory"
	} else if (info.Mode() & os.ModeSymlink) != 0 {
		fileType = "Symlink"
	} else if (info.Mode() & os.ModeNamedPipe) != 0 {
		fileType = "Named Pipe"
	} else if (info.Mode() & os.ModeSocket) != 0 {
		fileType = "Socket"
	} else if (info.Mode() & os.ModeDevice) != 0 {
		fileType = "Device"
	}

	permissions := info.Mode().String()
	owner := info.Sys().(*syscall.Stat_t).Uid
	group := info.Sys().(*syscall.Stat_t).Gid
	modTime := info.ModTime().Format(time.RFC1123)
	size := info.Size()

	details := fmt.Sprintf(
		"[yellow]Name:[white] %s\n[yellow]Type:[white] %s\n[yellow]Size:[white] %d bytes\n[yellow]Permissions:[white] %s\n[yellow]Owner:[white] %d\n[yellow]Group:[white] %d\n[yellow]Modified:[white] %s\n",
		info.Name(),
		fileType,
		size,
		permissions,
		owner,
		group,
		modTime,
	)
	fm.fileDetails.SetText(details)
}

func (fm *FileManager) deleteSelectedItem() {
	selectedIndex := fm.list.GetCurrentItem()
	if selectedIndex < 0 || selectedIndex >= len(fm.items) {
		return
	}

	selectedItem := fm.items[selectedIndex]
	fullPath := filepath.Join(fm.path, selectedItem)

	err := os.RemoveAll(fullPath) // Remove files or directories
	if err != nil {
		fm.fileDetails.SetText(fmt.Sprintf("Error deleting file: %v", err))
		return
	}

	fm.loadItems()
}

func (fm *FileManager) updateGitDetails() {
	if !isGitRepo(fm.path) {
		fm.gitDetails.SetText("")
		return
	}

	gitCommits := fm.gitLog()

	// Set Git details with commit hashes and messages
	fm.gitDetails.SetText(gitCommits)
}

func (fm *FileManager) gitLog() string {
	cmd := exec.Command("git", "-C", fm.path, "log", "--oneline", "-10")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error getting git log: %v", err)
	}

	// Process git log output to include commit hashes and messages
	lines := strings.Split(string(output), "\n")
	var result strings.Builder
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) >= 2 {
				result.WriteString(fmt.Sprintf("[yellow]%s [white]%s\n", parts[0], parts[1]))
			}
		}
	}
	return result.String()
}

func (fm *FileManager) run() {
	fm.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		fm.navigate(mainText)
	})

	fm.list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		fm.updateDetails()
		fm.updateGitDetails()
	})

	fm.commandInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			fm.commandMode = false
			fm.app.SetFocus(fm.list) // Return focus to list after exiting command mode
		} else if key == tcell.KeyEnter {
			// Process the command input
			command := fm.commandInput.GetText()
			fm.processCommand(command)
			fm.commandInput.SetText("") // Clear input field after processing command
		}
	})

	fm.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if fm.commandMode {
			return event
		}

		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyLeft:
			if fm.path != filepath.VolumeName(fm.path) && fm.path != "/" {
				fm.path = filepath.Dir(fm.path)
				fm.loadItems()
			}
			return nil
		case tcell.KeyEnter, tcell.KeyRight:
			index := fm.list.GetCurrentItem()
			if index >= 0 && index < len(fm.items) {
				fm.navigate(fm.items[index])
			}
			return nil
		case tcell.KeyCtrlD:
			fm.deleteSelectedItem()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case ':':
				fm.commandMode = true
				fm.app.SetFocus(fm.commandInput) // Switch focus to command input
				return nil
			case 'q':
				fm.app.Stop()
				return nil
			}
		}
		return event
	})

	fm.commands.SetText("[red]Navigate: [yellow]← Back, → Enter, [green]Ctrl+D Delete, [blue]: Command, [blue]q Quit")

	// Adjust layout to have the commands at the bottom center
	fm.flex.SetDirection(tview.FlexRow).
		AddItem(
							tview.NewFlex().
								SetDirection(tview.FlexColumn).
								AddItem(fm.list, 0, 1, true).
								AddItem(tview.NewTextView().SetText("").SetTextAlign(tview.AlignLeft), 1, 1, false).
								AddItem(fm.fileDetails, 0, 2, false).
								AddItem(tview.NewTextView().SetText("").SetTextAlign(tview.AlignLeft), 1, 1, false).
								AddItem(fm.gitDetails, 0, 2, false), 0, 1, true).
		AddItem(fm.commands, 3, 1, false).    // Use a height of 3 for the commands
		AddItem(fm.commandInput, 1, 1, false) // Add command input field at the bottom

	fm.app.SetRoot(fm.flex, true)

	// Set a dark black background color
	fm.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Clear()
		bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
		screen.SetStyle(bgStyle)
		return false
	})

	if err := fm.app.Run(); err != nil {
		panic(err)
	}
}

func (fm *FileManager) processCommand(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	parts := strings.Fields(command)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "mkdir":
		if len(args) < 1 {
			fm.printCommandOutput("Usage: mkdir <directory_name>")
			return
		}
		dirName := args[0]
		err := os.Mkdir(filepath.Join(fm.path, dirName), 0755)
		if err != nil {
			fm.printCommandOutput(fmt.Sprintf("Error creating directory: %v", err))
		} else {
			fm.printCommandOutput(fmt.Sprintf("Created directory '%s'", dirName))
			fm.loadItems()
		}
	case "touch":
		if len(args) < 1 {
			fm.printCommandOutput("Usage: touch <file_name>")
			return
		}
		fileName := args[0]
		filePath := filepath.Join(fm.path, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			fm.printCommandOutput(fmt.Sprintf("Error creating file: %v", err))
		} else {
			file.Close()
			fm.printCommandOutput(fmt.Sprintf("Created file '%s'", fileName))
			fm.loadItems()
		}
	case "ls":
		cmd := exec.Command("ls", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fm.printCommandOutput(fmt.Sprintf("Error running command: %v", err))
			return
		}
		fm.printCommandOutput(string(output))
	default:
		fm.printCommandOutput(fmt.Sprintf("Unknown command: %s", cmd))
	}
}

func (fm *FileManager) printCommandOutput(output string) {
	fm.fileDetails.SetText(output)
}

func main() {
    // Check if running as root (sudo)
    if os.Geteuid() != 0 {
        fmt.Println("This tool requires sudo privileges. Run with 'sudo'.")
        os.Exit(1)
    }

    fm := NewFileManager(".")
    fm.run()
}
