package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/controllers"
	"github.com/ovh/tat-contrib/al2tat/routes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mainCmd = &cobra.Command{
	Use:   "al2tat",
	Short: "Run Al2Tat",
	Long:  `Run Al2Tat`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("al2tat")
		viper.AutomaticEnv()

		if viper.GetBool("production") {
			// Only log the warning severity or above.
			log.SetLevel(log.WarnLevel)
			log.Info("Set Production Mode ON")
			gin.SetMode(gin.ReleaseMode)
		} else {
			log.SetLevel(log.DebugLevel)
		}

		if viper.GetString("log_level") != "" {
			switch viper.GetString("log_level") {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info":
				log.SetLevel(log.InfoLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			}
		}

		router := gin.Default()
		routes.InitRoutesAlerts(router)
		routes.InitRoutesMonitoring(router)
		routes.InitRoutesSystem(router)
		router.Run(":" + viper.GetString("listen_port"))
	},
}

var versionNewLine bool

// The version command prints this service.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of Al2Tat.",
	Run: func(cmd *cobra.Command, args []string) {
		if versionNewLine {
			fmt.Println(controllers.VERSION)
		} else {
			fmt.Print(controllers.VERSION)
		}
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionNewLine, "versionNewLine", "", true, "New line after version number")
	mainCmd.AddCommand(versionCmd)
	flags := mainCmd.Flags()
	flags.Bool("production", false, "Production mode")
	flags.String("log-level", "", "Log Level : debug, info or warn")
	flags.String("listen-port", "8082", "Tat Engine Listen Port")
	flags.String("url-tat-engine", "http://localhost:8080", "URL Tat Engine")
	viper.BindPFlag("production", flags.Lookup("production"))
	viper.BindPFlag("log_level", flags.Lookup("log-level"))
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))
	viper.BindPFlag("url_tat_engine", flags.Lookup("url-tat-engine"))
}

func main() {
	mainCmd.Execute()
}
