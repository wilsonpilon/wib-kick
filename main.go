package main

import (
	"fmt"
	"os"

	"github.com/rivo/tview"

	"sidekick/internal/tv"
)

func main() {
	app := tview.NewApplication()

	root := tv.NewApp(func() {
		app.Stop()
	})

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
