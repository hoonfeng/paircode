package main

import (
	"fmt"
	"os"

	"github.com/hoonfeng/paircode/internal/core"
)

func main() {
	wd, _ := os.Getwd()
	fmt.Println("cwd:", wd)
	fmt.Println("SettingsPath:", core.SettingsPath())
	core.Load()
	fmt.Println("loaded:", core.Loaded)
	fmt.Println("recentProjects:", core.Settings.RecentProjects)
	fmt.Println("workspaceFolders:", core.Settings.WorkspaceFolders)
	fmt.Println("lastProject:", core.Settings.LastProject)
	core.LoadLastProject()
	fmt.Println("after LoadLastProject recent:", core.Settings.RecentProjects)
	fmt.Println("Folders:", core.Folders)
}
