ECHO = @echo
GO = go

help:
	$(ECHO) "example-uservice app Makefile"
	$(ECHO) "'make go-build'	Build the app, don't forget to source your custom configuration in environment variables"
	$(ECHO) "'make go-test'		Install/Update go2xunit and start unit tests for Go files in the project"

go-build:
	$(GO) build -ldflags "-X main.Version=$(VERSION)"

go-test:
	# No tests currently because this is an example, delete the following line and replace it with:
	# $(GO) get -u github.com/tebeka/go2xunit
	# $(GO) test ./... -v | go2xunit -output tests_results.xml
	# once tests are implemented in the app
	$(GO) test ./... -v && touch tests_results.xml
