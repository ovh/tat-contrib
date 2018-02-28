ECHO = @echo
MAKE = make
API = api

help:
	$(ECHO) "tat2es app Makefile"
	$(ECHO) "'make go-build'	Build the app, don't forget to source your custom configuration in environment variables"
	$(ECHO) "'make go-test'		Install/Update go2xunit and start unit tests for Go files in the project"

go-build:
	$(MAKE) -C $(API) go-build

go-test:
	$(MAKE) -C $(API) go-test
