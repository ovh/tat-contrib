package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// VERSION is version of tat2xmpp.
const VERSION = "0.6.0"

var configFile string
var admins []string

var mainCmd = &cobra.Command{
	Use:   "tat2xmpp",
	Short: "Tat2XMPP",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("tat2xmpp")
		viper.AutomaticEnv()

		if viper.GetBool("production") {
			log.SetLevel(log.InfoLevel)
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

		router := gin.New()
		router.Use(recovery)

		if !viper.GetBool("production") {
			router.Use(gin.Logger())
		}

		router.Use(ginrus(log.StandardLogger(), time.RFC3339, true))

		router.Use(cors.Middleware(cors.Config{
			Origins:         "*",
			Methods:         "GET, PUT, POST, DELETE",
			RequestHeaders:  "Origin, Authorization, Content-Type, Accept, Tat_Password, Tat_Username",
			ExposedHeaders:  "Tat_Password, Tat_Username",
			MaxAge:          50 * time.Second,
			Credentials:     true,
			ValidateHeaders: false,
		}))

		initRoutes(router)

		s := &http.Server{
			Addr:           ":" + viper.GetString("listen_port"),
			Handler:        router,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Infof("tat2xmpp running on %s", viper.GetString("listen_port"))

		readConfigFile()

		var err error
		tatbot, err = getBotClient(viper.GetString("username_tat_engine"), viper.GetString("password_tat_engine"))
		if err != nil {
			log.Fatalf("Error while initialize client err:%s", err)
		}

		admins = strings.Split(viper.GetString("admin_tat2xmpp"), ",")
		log.Infof("admin configured:%+v", admins)

		go tatbot.born()

		if err := s.ListenAndServe(); err != nil {
			log.Errorf("Error while running ListenAndServe: %s", err.Error())
		}
	},
}

func init() {

	flags := mainCmd.Flags()
	flags.Bool("production", false, "Production mode")
	viper.BindPFlag("production", flags.Lookup("production"))

	flags.String("log-level", "", "Log Level : debug, info or warn")
	viper.BindPFlag("log_level", flags.Lookup("log-level"))

	flags.String("listen-port", "8088", "Tat2XMPP Listen Port")
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))

	flags.String("hook-key", "", "Hook Key, for using POST http://<url>/hook endpoint, with Header TAT2XMPPKEY ")
	viper.BindPFlag("hook_key", flags.Lookup("hook-key"))

	flags.String("url-tat-engine", "http://localhost:8080", "URL Tat Engine")
	viper.BindPFlag("url_tat_engine", flags.Lookup("url-tat-engine"))

	flags.String("username-tat-engine", "tat.system.xmpp", "Username Tat Engine")
	viper.BindPFlag("username_tat_engine", flags.Lookup("username-tat-engine"))

	flags.String("password-tat-engine", "", "Password Tat Engine")
	viper.BindPFlag("password_tat_engine", flags.Lookup("password-tat-engine"))

	flags.String("xmpp-server", "", "XMPP Server")
	viper.BindPFlag("xmpp_server", flags.Lookup("xmpp-server"))

	flags.String("xmpp-bot-jid", "tat@localhost", "XMPP Bot JID")
	viper.BindPFlag("xmpp_bot_jid", flags.Lookup("xmpp-bot-jid"))

	flags.String("xmpp-bot-password", "", "XMPP Bot Password")
	viper.BindPFlag("xmpp_bot_password", flags.Lookup("xmpp-bot-password"))

	flags.String("xmpp-hello-world", "", "Sending Hello World message to this jabber id")
	viper.BindPFlag("xmpp_hello_world", flags.Lookup("xmpp-hello-world"))

	flags.String("admin-tat2xmpp", "", "Admin tat2xmpp admina@jabber.xxx.net,adminb@jabber.xxx.net,")
	viper.BindPFlag("admin_tat2xmpp", flags.Lookup("admin-tat2xmpp"))

	flags.String("url-tatwebui", "", "TatwebUI base URL")
	viper.BindPFlag("url_tatwebui", flags.Lookup("url-tatwebui"))

	flags.String("more-help", "", "Text added on /tat help")
	viper.BindPFlag("more_help", flags.Lookup("more-help"))

	flags.Bool("xmpp-debug", false, "XMPP Debug")
	viper.BindPFlag("xmpp_debug", flags.Lookup("xmpp-debug"))

	flags.Bool("xmpp-notls", true, "XMPP No TLS")
	viper.BindPFlag("xmpp_notls", flags.Lookup("xmpp-notls"))

	flags.Bool("xmpp-starttls", false, "XMPP Start TLS")
	viper.BindPFlag("xmpp_starttls", flags.Lookup("xmpp-starttls"))

	flags.Bool("xmpp-session", true, "XMPP Session")
	viper.BindPFlag("xmpp_session", flags.Lookup("xmpp-session"))

	flags.Bool("xmpp-insecure-skip-verify", true, "XMPP InsecureSkipVerify")
	viper.BindPFlag("xmpp_insecure_skip_verify", flags.Lookup("xmpp-insecure-skip-verify"))

	flags.Int("xmpp-delay", 5, "Delay between two sent messages")
	viper.BindPFlag("xmpp_delay", flags.Lookup("xmpp-delay"))

	mainCmd.PersistentFlags().StringVarP(&configFile, "configFile", "c", "", "configuration file")
}

func ginrus(l *log.Logger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		sec := latency.Seconds()
		ms := int64(latency / time.Millisecond)

		entry := l.WithFields(log.Fields{
			"status":                  c.Writer.Status(),
			"method":                  c.Request.Method,
			"path":                    path,
			"query":                   query,
			"ip":                      c.ClientIP(),
			"latency":                 latency,
			"latency_nanosecond_int":  latency.Nanoseconds(),
			"latency_millisecond_int": ms,
			"latency_second_float":    sec,
			"user-agent":              c.Request.UserAgent(),
			"time":                    end.Format(timeFormat),
		})

		msg := fmt.Sprintf("%d %s %s %fs %dms %dns", c.Writer.Status(), c.Request.Method, path, sec, ms, latency)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			entry.Error(fmt.Sprintf("ERROR %s %s", msg, c.Errors.String()))
		} else if c.Writer.Status() >= 400 {
			entry.Warn(fmt.Sprintf("WARN %s", msg))
		} else if path != "/version" {
			entry.Info(fmt.Sprintf("INFO %s", msg))
		}
	}
}

// recovery is a middleware that recovers from any panics and writes a 500 if there was one.
func recovery(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			path := c.Request.URL.Path
			query := c.Request.URL.RawQuery
			trace := make([]byte, 4096)
			count := runtime.Stack(trace, true)
			log.Errorf("[recovery] err:%s method:%s path:%s query:%s stacktrace of %d bytes:%s",
				err, c.Request.Method, path, query, count, trace)

			c.AbortWithStatus(500)
		}
	}()
	c.Next()
}

func main() {
	mainCmd.Execute()
}
