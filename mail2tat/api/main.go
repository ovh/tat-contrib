package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	log "github.com/Sirupsen/logrus"
)

const VERSION = "1.0.0"

var mainCmd = &cobra.Command{
	Use:   "mail2tat",
	Short: "MAIL2TAT - Mail to Tat",
	Long:  `MAIL2TAT - Mail to Tat`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("mail2tat")
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
		initRoutes(router)
		initInstance()

		if viper.GetBool("activate_cron") {
			initAndStartCron()
		} else {
			log.Warn("Cron is disabled by flag --activate-cron")
		}

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Infof("Running on %s", viper.GetString("listen_port"))

		if err := s.ListenAndServe(); err != nil {
			log.Info("Error while running ListenAndServe: %s", err.Error())
		}
	},
}

func init() {
	flags := mainCmd.Flags()
	flags.Bool("production", false, "Production mode")
	viper.BindPFlag("production", flags.Lookup("production"))

	flags.String("log-level", "", "Log Level : debug, info or warn")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("listen-port", "8084", "RunKPI Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("url-tat-engine", "http://localhost:8080", "URL Tat Engine")
	viper.BindPFlag("url_tat_engine", flags.Lookup("url-tat-engine"))

	flags.String("username-tat-engine", "", "Username Tat Engine")
	viper.BindPFlag("username_tat_engine", flags.Lookup("username-tat-engine"))

	flags.String("password-tat-engine", "", "Password Tat Engine")
	viper.BindPFlag("password_tat_engine", flags.Lookup("password-tat-engine"))

	flags.String("allowed-domains", "", "Allowed from domains. Empty: no-restriction. Ex: --allowed-domains=domainA.org,domainA.com")
	viper.BindPFlag("allowed_domains", flags.Lookup("allowed-domains"))

	flags.String("imap-host", "", "IMAP Host")
	viper.BindPFlag("imap_host", flags.Lookup("imap-host"))

	flags.String("imap-username", "", "IMAP Username")
	viper.BindPFlag("imap_username", flags.Lookup("imap-username"))

	flags.String("imap-password", "", "IMAP Password")
	viper.BindPFlag("imap_password", flags.Lookup("imap-password"))

	flags.Bool("activate-cron", true, "Activate Cron")
	viper.BindPFlag("activate_cron", flags.Lookup("activate-cron"))
}

func main() {
	mainCmd.Execute()
}
