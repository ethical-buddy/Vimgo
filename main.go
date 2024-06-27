package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FileManager struct {
	app        *tview.Application
	path       string
	fileList   *tview.List
	vimOutput  *tview.TextView
	currentDir string
}

func NewFileManager(path string) *FileManager {
	app := tview.NewApplication()
	fileList := tview.NewList().
		ShowSecondaryText(false).
		SetSelectedBackgroundColor(tcell.ColorLightBlue).
		SetSelectedTextColor(tcell.ColorBlack).
		SetBorder(true).
		SetTitle(" File Manager ")

	vimOutput := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	return &FileManager{
		app:        app,
		path:       path,
		fileList:   fileList,
		vimOutput:  vimOutput,
		currentDir: path,
	}
}

func (fm *FileManager) run() error {
	if err := fm.refreshFileList(); err != nil {
		return err
	}

	fm.fileList.SetInputCapture(fm.handleInput)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fm.fileList, 0, 1, true).
		AddItem(fm.vimOutput, 0, 2, false)

	if err := fm.app.SetRoot(flex, true).Run(); err != nil {
		return err
	}
	return nil
}

func (fm *FileManager) refreshFileList() error {
	fm.fileList.Clear()

	entries, err := os.ReadDir(fm.path)
	if err != nil {
		return err
	}

	var fileNames []string
	for _, entry := range entries {
		fileNames = append(fileNames, entry.Name())
	}

	sort.Strings(fileNames)

	for _, name := range fileNames {
		fm.fileList.AddItem(name, "", 0, nil)
	}

	return nil
}

func (fm *FileManager) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		fm.handleEnter()
	case tcell.KeyCtrlL:
		fm.app.Draw()
	case tcell.KeyCtrlC, tcell.KeyEsc:
		fm.app.Stop()
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q':
			fm.quitVim()
		}
	}

	return event
}

func (fm *FileManager) handleEnter() {
	selectedItemIndex := fm.fileList.GetCurrentItem()
	if selectedItemIndex < 0 || selectedItemIndex >= fm.fileList.GetItemCount() {
		return
	}

	selectedItem := fm.fileList.GetItemAt(selectedItemIndex)
	if selectedItem == nil {
		return
	}

	filePath := filepath.Join(fm.path, selectedItem.GetText())

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Fprintf(fm.vimOutput, "Error: %s\n", err)
		return
	}

	if fileInfo.IsDir() {
		fm.path = filePath
		if err := fm.refreshFileList(); err != nil {
			fmt.Fprintf(fm.vimOutput, "Error: %s\n", err)
		}
	} else {
		fm.openInVim(filePath)
	}
}

func (fm *FileManager) openInVim(filePath string) {
	fm.vimOutput.Clear()

	cmd := exec.Command("vim", filePath)
	cmd.Stdout = fm.vimOutput
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(fm.vimOutput, "Error: %s\n", err)
		return
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(fm.vimOutput, "Error: %s\n", err)
	}
}

func (fm *FileManager) quitVim() {
	// Placeholder for any cleanup needed when quitting Vim
}

func main() {
	path, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %s\n", err)
		os.Exit(1)
	}

	fm := NewFileManager(path)
	if err := fm.run(); err != nil {
		fmt.Printf("Error running file manager: %s\n", err)
		os.Exit(1)
	}
}
