ECHO = @echo
GO = go

help:
	$(ECHO) "al2tat app Makefile"
	$(ECHO) "'make go-build'	Build the app, don't forget to source your custom configuration in environment variables"

go-build:
	$(GO) build -ldflags "-X github.com/ovh/tat-contrib/al2tat/controllers.Version=$(VERSION)"
