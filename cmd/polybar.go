// Copyright © 2018 Patrick Motard <motard19@gmail.com>
// TODO: add i3wm dimensions to bar settings

package cmd

import (
	"bufio"
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type display struct {
	name string // example: DP-4 or HDMI-1
	// Position is where the display is relative to other displays on the screen.
	// Screens are comprised of one or more displays.
	xposition int16  // the x coordinate of the display on the screen
	yposition int16  // the y coordinate of the display on the screen
	xres      uint16 // The ideal x resolution.
	yres      uint16 // The idea y resolution.
	primary   bool   // Whether or not the display is the primary (main) display.
	active    bool
}

var (
	_theme                 string
	InstalledPolybarThemes []string
	FullThemePath          string
	FullThemesPath         string
)

var polybarCmd = &cobra.Command{
	Use:   "polybar",
	Short: "Loads polybar themes and bars.",
	Long:  "TODO: add long description",
	Run: func(cmd *cobra.Command, args []string) {
		FullThemesPath = Home + "/" + Config.Polybar.ThemesDirectory
		FullThemePath = FullThemesPath + "/" + _theme + "/config"
		validateTheme()
		if viper.GetBool("list") == true {
			listThemes()
			os.Exit(0)
		}
		main()
	},
}

func init() {
	polybarCmd.Flags().StringVarP(&_theme, "theme", "t", "", "Load a Polybar theme by name. The theme specified will be saved to dot's current_settings.")
	polybarCmd.Flags().BoolP("list", "l", false, "Lists all themes found on the system.")
	viper.BindPFlag("polybar.theme", polybarCmd.Flags().Lookup("theme"))
	// TODO: This is putting list on viper, which is then written to file
	// either figure out how to unmarshal Config and overwrite current settings with it,
	// or figure out how to reference flags without viper. 'list=true' doesn't belong in current_settings
	viper.BindPFlag("list", polybarCmd.Flags().Lookup("list"))
	rootCmd.AddCommand(polybarCmd)
}

func listThemes() {
	for _, t := range InstalledPolybarThemes {
		fmt.Println(t)
	}
}

func validateTheme() {
	if Config.Polybar.ThemesDirectory == "" {
		log.Fatalln("Please set polybar.themes_directory in current_settings.yml")
	}
	// TODO: validate theme in cs (current_settings.yml) exists on filesystem
	// TODO: if theme passed in exists and is different than the current theme, write it to cs

	// Look up installed themes.
	// A theme is considered to be installed if there is a directory with the themes name,
	// in the themes folder.
	viper.Set("polybar.theme", _theme)
	Config.Polybar.Theme = _theme
	t := false

	f, err := ioutil.ReadDir(FullThemesPath)
	if err != nil {
		log.Errorln(err)
		log.Fatalf("Failed to read themes from %s", FullThemesPath)
	}

	for _, x := range f {
		if x.IsDir() && x.Name() != "global" {
			if _theme == x.Name() {
				t = true
			}
			InstalledPolybarThemes = append(InstalledPolybarThemes, x.Name())
		}
	}
	if t {
		viper.WriteConfig()
	}
}

func main() {
	// connect to X server
	X, _ := xgb.NewConn()
	err := randr.Init(X)
	if err != nil {
		log.Fatal(err)
	}

	// get root node
	root := xproto.Setup(X).DefaultScreen(X).Root
	// get the resources of the screen
	resources, err := randr.GetScreenResources(X, root).Reply()
	if err != nil {
		log.Fatal(err)
	}
	// get the primary output
	primaryOutput, _ := randr.GetOutputPrimary(X, root).Reply()

	var displays []display
	// go through the connected outputs and get their position and resolution
	for _, output := range resources.Outputs {
		info, err := randr.GetOutputInfo(X, output, 0).Reply()
		if err != nil {
			log.Fatal(err)
		}
		if info.Connection == randr.ConnectionConnected {
			d := display{
				name: string(info.Name),
			}
			crtc, err := randr.GetCrtcInfo(X, info.Crtc, 0).Reply()
			if err != nil {
				// log.Fatal("Failed to get CRTC info", err)
				// "BadCrtc" happens when attempting to get
				// a crtc for an output is disabled (inactive).
				// TODO: figure out a better way to identify active vs inactive
				d.active = false
			} else {
				d.active = true
				d.xposition = crtc.X
				d.yposition = crtc.Y
			}

			if output == primaryOutput.Output {
				d.primary = true
			} else {
				d.primary = false
			}
			bestMode := info.Modes[0]
			for _, mode := range resources.Modes {
				if mode.Id == uint32(bestMode) {
					d.xres = mode.Width
					d.yres = mode.Height
				}
			}
			displays = append(displays, d)
		}
	}

	// order the displays by their position, left to right.
	sort.Slice(displays, func(i, j int) bool {
		return displays[i].xposition < displays[j].xposition
	})

	// kill all polybar sessions polybar
	cmd := exec.Command("sh", "-c", "killall -q polybar")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// TODO: handle case where polybar isn't running yet
		// log.Fatalln("Failed to kill polybar")
		log.Println("Failed to kill polybar")
	}
	fmt.Println(string(out))

	// create the env vars we'll hand to polybar
	// polybar needs to know the theme, and what the left, right and main monitor are
	var polybarEnvVars []string
	for i, d := range displays {
		// skip inactive monitors
		if !d.active {
			continue
		}
		if d.primary {
			s := fmt.Sprintf("MONITOR_MAIN=%s", d.name)
			polybarEnvVars = append(polybarEnvVars, s)
		} else if i == 0 {
			s := fmt.Sprintf("MONITOR_LEFT=%s", d.name)
			polybarEnvVars = append(polybarEnvVars, s)
		} else if i == 1 || i == 2 {
			s := fmt.Sprintf("MONITOR_RIGHT=%s", d.name)
			polybarEnvVars = append(polybarEnvVars, s)
		}
	}
	// add the theme to the environment
	t := fmt.Sprintf("polybar_theme=%s", FullThemePath)
	log.Infoln(t)
	polybarEnvVars = append(polybarEnvVars, t)

	// create a new array of env vars, appending the current environment
	// with the env vars created above
	newEnv := append(os.Environ(), polybarEnvVars...)

	// get the theme object for current theme from current_settings
	var theme Theme
	// TODO: maybe switch Themes to a map so i don't have to loop
	for _, t := range Config.Polybar.Themes {
		if Config.Polybar.Theme == t.Name {
			theme = t
		}
	}
	// Exit if it fails to find the theme object
	if theme.Name == "" {
		log.Error("Theme not found, exiting.")
		os.Exit(1)
	}

	// load bars from theme's .rasi file if none were specified in current_settings.yml
	var bars []string
	if len(theme.Bars) == 0 {
		log.Infoln("No bars specified in current-settings file. Auto-detecting bars...")
		bars = getBars(theme, FullThemePath)
	} else {
		log.Infoln("Bars specified in current-settings file.")
		bars = theme.Bars
	}
	// start all the bars
	for _, bar := range bars {
		log.Infoln(fmt.Sprintf("Loading bar '%s'", bar))
		polybar(newEnv, bar)
	}
}

func getBars(theme Theme, path string) []string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var b []string
	scanner := bufio.NewScanner(f)
	// example: [bar/SOME.BAR] -> SOME.BAR
	re := regexp.MustCompile(`^\[bar\/(.*?)\]`)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "[bar/") {
			m := re.FindSubmatch(scanner.Bytes())
			b = append(b, string(m[1]))
		}
	}
	if len(b) == 0 {
		// TODO use a public variable to reference the path to current_settings.yml
		log.Fatalf("No bars found in:\n - %s\n - %s", FullThemePath, Home+"/code/dot/current_settings.yml")
	}
	return b
}

func polybar(env []string, bar string) string {
	s := fmt.Sprintf("polybar -r %s", bar)
	cmd := exec.Command("bash", "-c", s)
	cmd.Env = env
	cmd.Start()
	return fmt.Sprintf("Finished bar %s", bar)
}
