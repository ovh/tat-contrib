ECHO = @echo
GO = go

help:
	$(ECHO) "al2tat app Makefile"
	$(ECHO) "'make go-build'	Build the app, don't forget to source your custom configuration in environment variables"
	$(ECHO) "'make go-test'		Install/Update go2xunit and start unit tests for Go files in the project"

go-build:
	$(GO) build -ldflags "-X github.com/ovh/tat-contrib/al2tat/controllers.Version=$(VERSION)"

go-test:
	$(GO) get -u github.com/tebeka/go2xunit
	$(GO) test ./... -v | go2xunit -output tests_results.xml
