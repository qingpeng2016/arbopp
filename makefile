.PHONY: build
build: ## Build the fgmm binary.
	go build -o fgmm -buildvcs=false .
	@echo "structure server build successful"
