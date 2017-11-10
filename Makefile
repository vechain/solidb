PACKAGE = github.com/vechain/solidb
TARGET = bin/solidb
SYS_GOPATH := $(GOPATH)
GOPATH = $(CURDIR)/.gopath
BASE = $(GOPATH)/src/$(PACKAGE)

DATEVERSION=`date -u +%Y%m%d`
COMMIT=`git --no-pager log --pretty="%h" -n 1`

.PHONY: all
all: |$(BASE)
	@cd $(BASE) && go build -o $(TARGET) -ldflags "-X main.version=${DATEVERSION} -X main.gitCommit=${COMMIT}"	

$(BASE):
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

.PHONY: install
install: all
	@mv $(TARGET) $(SYS_GOPATH)/bin/

.PHONY: clean
clean:
	-rm -f $(TARGET)