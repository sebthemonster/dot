// Copyright © 2018 Patrick Motard <motard19@gmail.com>

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// portCmd represents the port command
var portCmd = &cobra.Command{
	Use:   "port",
	Short: "set/get alsa port id",
	Long: `This is used by ~/.local/bin/tools/polybar_alsa_module.
If you give no arguements, it returns the current port id.
If you pass in a port, it sets config file to the port.
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// example of setting a value and writing config:
		// var newKeypair = make(map[string]string)
		// newKeypair["test"] = "val"
		// viper.Set("outerKey", newKeypair)
		// viper.WriteConfig()
		if len(args) == 0 {
			// 	fmt.Println(settings.Sound.Port)
			// 	os.Exit(0)
			// fmt.Println("hello")
			fmt.Println(viper.Get("sound.port"))
			os.Exit(0)
		}
		viper.Set("sound.port", args[0])
		viper.WriteConfig()

		// fmt.Println(viper.Get("sound.port"))
		// settings.Sound.Port = args[0]
		// settings.WriteSettings()
	},
}

func init() {
	soundCmd.AddCommand(portCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// portCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// portCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
