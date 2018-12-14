package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version of tat2es
var Version string

var mainCmd = &cobra.Command{
	Use:   "tat2es",
	Short: "TAT To ElasticSearch",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("tat2es")
		viper.AutomaticEnv()

		if viper.GetBool("production") {
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

		router.Use(cors.Middleware(cors.Config{
			Origins:         "*",
			Methods:         "GET, PUT, POST, DELETE",
			RequestHeaders:  "Origin, Authorization, Content-Type, Accept, Tat_Password, Tat_Username",
			ExposedHeaders:  "Tat_Password, Tat_Username",
			MaxAge:          50 * time.Second,
			Credentials:     true,
			ValidateHeaders: false,
		}))

		router.GET("/version", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"version": Version})
		})

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		if len(strings.Split(viper.GetString("topics_indexes"), ",")) == 0 {
			log.Errorf("Invalid argument, --topics-indexes is empty. See help.")
			os.Exit(1)
		}

		conns, err := getClientsES()
		if err != nil {
			log.Fatal(err)
		}

		var chans []chan<- *indexableData
		for _, conn := range conns {
			postESChan := make(chan *indexableData)
			chans = append(chans, postESChan)
			go postES(conn, postESChan)
		}
		go do(chans)

		log.Infof("RRunning on %s", viper.GetString("listen_port"))
		if err := s.ListenAndServe(); err != nil {
			log.Errorf("Error while running ListenAndServe: %s", err.Error())
		}
	},
}

// The version command prints this service.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of tat2es.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(Version)
	},
}

func init() {
	mainCmd.AddCommand(versionCmd)

	flags := mainCmd.Flags()
	flags.Bool("production", false, "Production mode")
	viper.BindPFlag("production", flags.Lookup("production"))

	flags.String("log-level", "", "Log Level: debug, info or warn")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("listen-port", "8086", "Tat2ES Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("url-tat-engine", "http://localhost:8080", "URL Tat Engine")
	viper.BindPFlag("url_tat_engine", flags.Lookup("url-tat-engine"))

	flags.String("username-tat-engine", "", "Username Tat Engine")
	viper.BindPFlag("username_tat_engine", flags.Lookup("username-tat-engine"))

	flags.String("password-tat-engine", "", "Password Tat Engine")
	viper.BindPFlag("password_tat_engine", flags.Lookup("password-tat-engine"))

	flags.Int("pause-tat", 5, "Pause in second after each call on tat for each topic")
	viper.BindPFlag("pause_tat", flags.Lookup("pause-tat"))

	flags.String("protocol-es", "http", "Protocol ElasticSearch http or https")
	viper.BindPFlag("protocol_es", flags.Lookup("protocol-es"))

	flags.String("host-es", "", "Host ElasticSearch")
	viper.BindPFlag("host_es", flags.Lookup("host-es"))

	flags.String("user-es", "", "User ElasticSearch")
	viper.BindPFlag("user_es", flags.Lookup("user-es"))

	flags.String("password-es", "", "Password ElasticSearch")
	viper.BindPFlag("password_es", flags.Lookup("password-es"))

	flags.String("pause-es", "", "Pause in ms after each send on ES")
	viper.BindPFlag("pause_es", flags.Lookup("pause-es"))

	flags.String("port-es", "9200", "Port ElasticSearch")
	viper.BindPFlag("port_es", flags.Lookup("port-es"))

	flags.String("force-index-es", "", "Force ES index")
	viper.BindPFlag("force_index_es", flags.Lookup("force-index-es"))

	flags.String("cron-schedule", "@every 3h", "Cron Schedule, see https://godoc.org/github.com/robfig/cron")
	viper.BindPFlag("cron_schedule", flags.Lookup("cron-schedule"))

	flags.String("topics-indexes", "", "/Topic/Sub-Topic1:ES_Index1,/Topic/Sub-Topic2:ES_Index2")
	viper.BindPFlag("topics_indexes", flags.Lookup("topics-indexes"))

	flags.Int("last-hour-min-creation", 168, "Index all messages since <last-hour-min-creation> hours")
	viper.BindPFlag("last_hour_min_creation", flags.Lookup("last-hour-min-creation"))

	flags.Int("messages-limit", 50, "messages-limit is used by MessageCriteria.Limit for requesting TAT")
	viper.BindPFlag("messages_limit", flags.Lookup("messages-limit"))
}

func main() {
	mainCmd.Execute()
}
