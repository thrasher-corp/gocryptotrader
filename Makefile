GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)$(SLASH)/bin
MAINFOLDER := /gocryptotrader_filesystem
INSTALLDIR := $(GOBIN)$(MAINFOLDER)
INSTALLDBFILE := $(INSTALLDIR)/database.db
BUILDDIR := .$(MAINFOLDER)
BUILDDBFILE := $(INSTALLDIR)/database.db

get:
	# Ensures that all dependancies at the correct version/branch etc
	dep ensure

build:
	# Builds in this currency directory
	@echo MAKE BUILD - GoCryptoTrader
	dep ensure
ifeq ("$(wildcard $(BUILDDIR))","")
	@echo MAKE BUILD - GoCryptoTrader filesystem added
	@mkdir $(BUILDDIR)
else
	@echo MAKE BUILD - GoCryptoTrader filesystem already found
endif

ifeq ("$(wildcard $(BUILDDBFILE))","")
	@echo MAKE BUILD - Copying database
	@cp ./database/database.db $(BUILDDIR)
else
	@echo MAKE BUILD - Old database found, flushing
	@rm $(BUILDDBFILE)
	@cp ./database/database.db $(BUILDDIR)
endif

	go build .
	@echo MAKE BUILD - GoCryptoTrader succesfully built in current directory

install:
	# Builds @ GOPATH
	@echo MAKE INSTALL - GoCryptoTrader to GOPATH: $(GOPATH)
	dep ensure
ifeq ("$(wildcard $(INSTALLDIR))","")
	@echo MAKE INSTALL - GoCryptoTrader filesystem added
	@mkdir $(INSTALLDIR)
else
	@echo MAKE INSTALL - GoCryptoTrader filesystem already found
endif

ifeq ("$(wildcard $(INSTALLDBFILE))","")
	@echo MAKE INSTALL - Copying database
	@cp ./database/database.db $(INSTALLDIR)
else
	@echo MAKE INSTALL - Old database found, flushing
	@rm $(INSTALLDBFILE)
	@cp ./database/database.db $(INSTALLDIR)
endif

	go install
	@echo MAKE INSTALL - GoCryptoTrader succesfully built in $(GOPATH)
