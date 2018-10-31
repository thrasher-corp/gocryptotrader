GOPATH := $(shell go env GOPATH)

a:
	@echo GoCryptoTrader - Please ensure you use either 'updatedep', 'build' or 'install' tags for make command

updatedep:
	@echo GoCryptoTrader - Ensuring and updating vendor dependancy list...
	@dep ensure -update

build:
	@echo GoCryptoTrader - Building...
	@dep ensure
	@go build . 
	@echo GoCryptoTrader - Succesfully built in current directory

install:
ifeq ($(OS),Windows_NT)    
	@echo GoCryptoTrader - Building to GOPATH: $(GOPATH)\bin
else
	@echo GoCryptoTrader - Building to GOPATH: $(GOPATH)/bin
endif

	@dep ensure
	@go install

ifeq ($(OS),Windows_NT)
	@echo GoCryptoTrader - Succesfully built in $(GOPATH)\bin
else
	@echo GoCryptoTrader - Succesfully built in $(GOPATH)/bin
endif