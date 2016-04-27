// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"
)

var programCmd = &cobra.Command{
	Use:   "program",
	Short: "Searches current directory for DNA flags",
	Long: `Searches the current directory for any lines with @@DNA:[function name]
  @@END. It will give you the quick description and a way to submit your code`,
	Run: Enter,
}

// The global variable `matches` contains all of the matches for @@DNA...@@END in the specified
// folder
var matches = make([]DNAFile, 0)

type DNAFile struct {
	Location    string
	Description string
	Status      string
	Input       string
	Output      string
	Place       int
}

func Enter(cmd *cobra.Command, args []string) {
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		fmt.Println(err)
	}
	defer g.Close()

	// Set matches equal to everything in the current directory
	var target_directory string
	if len(args) == 1 {
		target_directory = args[0]
	} else {
		target_directory = "."
	}

	files := make([]string, 0)
	err := ListFiles(target_directory, &files)
	if err != nil {
		fmt.Println(err)
	}

	err = GetMatches(&files)
	if err != nil {
		fmt.Println(err)
	}

	// Finish GUI

	g.SetLayout(Layout)
	if err := KeyBindings(g); err != nil {
		log.Panic(err)
	}
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		// handle error
	}

}

/*
@@DNA
Description: The main screen for the GoCUI Layout. It needs to set up the layout screen with a legend of how to use DNA's front-end, and then call NewView to create the first View. The position of the legend should be relative to the screen size by 1/5 in the upper right-hand corner.
Input: g is a pointer to GoCUI's GUI. It can be used to create the view. Check the GoDoc for GoCUI to learn how to use it: https://godoc.org/github.com/jroimartin/gocui
Output: error is returned if there are any errors when making the layout
Status: Done
@@END
*/
func Layout(g *gocui.Gui) error {
	maxX, _ := g.Size()
	if v, err := g.SetView("main", 4*maxX/5, 0, maxX-1, 4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		fmt.Fprintf(v, "← → to navigate\nCtrl-C to exit\n'v' to open in vim")
		err := NewView(g)
		if err != nil {
			return err
		}
	}

	return nil
}

var viewIndex = 0

/*
Description: NewView creates a new window that takes up 4/5 of the screen in the X direction, and all of the screen in the Y direction. It takes the viewIndex global variable to determine what element in the `matches` array it must select. In the new view, a prettified output of the DNAFile struct is added.
Input: g is a pointer to GoCUI's GUI. To create a new view, read the doc for GoCUI under https://godoc.org/github.com/jroimartin/gocui#Gui.SetView
Output: error is retuned if there are any errors along the way, or nil if there are no errors
Status: Done
*/
func NewView(g *gocui.Gui) error {
	if viewIndex >= len(matches) {
		viewIndex = len(matches) - 1
	}
	if viewIndex < 0 {
		viewIndex = 0
	}
	var match = matches[viewIndex]
	maxX, maxY := g.Size()

	v, err := g.SetView(strconv.Itoa(viewIndex), 0, 0, 4*maxX/5, maxY-1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = true
		if match.Location != "" {
			fmt.Fprintf(v, "%s\n", match.Location)
		}
		if match.Description != "" {
			fmt.Fprintf(v, "Description:\n%s\n", match.Description)
		}
		if match.Input != "" {
			fmt.Fprintf(v, "Input:\n%s\n", match.Input)
		}
		if match.Output != "" {
			fmt.Fprintf(v, "Output:\n%s\n", match.Output)
		}
		if match.Status != "" {
			fmt.Fprintf(v, "Status:%s\n", match.Status)
		}
		fmt.Fprintf(v, "\n%d/%d\n", viewIndex+1, len(matches))
	}
	_, err = g.SetViewOnTop("main")
	if err != nil {
		return err
	}
	if err := g.SetCurrentView(strconv.Itoa(viewIndex)); err != nil {
		return err
	}
	if viewIndex-1 >= 0 {
		if err != g.DeleteView(strconv.Itoa(viewIndex-1)) {
			return err
		}
	}
	return nil
}

func Quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func OpenInEditor(g *gocui.Gui, v *gocui.View) error {
	cmd := exec.Command("vim", matches[viewIndex].Location)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	// Bug - Going back to DNA from vim breaks DNA
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func NextView(g *gocui.Gui, v *gocui.View) error {
	viewIndex += 1
	return NewView(g)
}

func PreviousView(g *gocui.Gui, v *gocui.View) error {
	viewIndex -= 1
	return NewView(g)
}

func KeyBindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, Quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'v', gocui.ModNone, OpenInEditor); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, NextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, PreviousView); err != nil {
		return err
	}
	return nil
}

/*
@@DNA
Description: Gived a list of files, it searhes each file for a string containing the keyword (@)@DNA and (@)@END. It then places the matches in an array of DNAFiles, which is a struct specified in this file.
Input: files is a string array pointer that contains all files (not directories) currently selected to be searched.
Output: err is returned with any errors encountered in the process of getting matches
Status: Done
@@END
*/
func GetMatches(files *[]string) (err error) {
	for _, element := range *files {
		file_data, err := ioutil.ReadFile(element)
		if err != nil {
			fmt.Println(err)
		}
		if element[len(element)-5:len(element)] == ".java" || element[len(element)-3:len(element)] == ".js" {
			replacer := strings.NewReplacer("//", "")
			file_data = []byte(replacer.Replace(string(file_data)))
		}
		if element[len(element)-3:len(element)] == ".py" {
			replacer := strings.NewReplacer("#", "")
			file_data = []byte(replacer.Replace(string(file_data)))
		}
		// There is a [@] so that DNA doesn't catch it while running this program
		re := regexp.MustCompile("(?s)@[@]DNA(.*?)@@END")
		matches_raw := re.FindAllString(string(file_data), -1)
		for _, match := range matches_raw {
			var matchSplit []string

			if len(match) != 0 {
				matchSplit = strings.Split(string(match), "\n")
				var tmpDNA DNAFile
				for _, m := range matchSplit {
					m = strings.TrimSpace(m)
					// Check which line follows which tag
					switch {
					case m == "@@DNA" || m == "@@END":
						continue
					case strings.Contains(m, "Status: "):
						tmpDNA.Status = m[7:len(m)]
					case strings.Contains(m, "Input: "):
						tmpDNA.Input = m[6:len(m)]
					case strings.Contains(m, "Output: "):
						tmpDNA.Output = m[7:len(m)]
					case strings.Contains(m, "Description: "):
						tmpDNA.Description = m[12:len(m)]
					}

				}
				tmpDNA.Location = element
				if tmpDNA.Description != "" {
					matches = append(matches, tmpDNA)
				}

			}
		}
	}
	return
}

/*
@@DNA
Description: Directory walks the specified directory and adds all of the selected files (excluding directories) to a string array passed by refrence
Input: target_directory is the specified directory. It can be a full path or relative path. files is a pointer to a string array. This is where the output of the walk needs to be placed
Output: err is what needs to be returned if there are any errors
Status: Done
@@END
*/
func ListFiles(target_directory string, files *[]string) (err error) {
	err = filepath.Walk(target_directory, func(path string, _ os.FileInfo, _ error) (err error) {
		// Get status on target path so that we can make sure it's not a directory
		file_stat, err := os.Stat(path)

		if !file_stat.IsDir() {
			*files = append(*files, path)
		}
		return

	})

	return
}

func init() {
	RootCmd.AddCommand(programCmd)
}
