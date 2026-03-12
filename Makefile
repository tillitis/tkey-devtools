# Check for OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	BUILD_CGO_ENABLED ?= 1
else
	BUILD_CGO_ENABLED ?= 0
endif

.PHONY: all
all: tkey-runapp

DESTDIR=/
PREFIX=/usr/local
UDEVDIR=/etc/udev
destbin=$(DESTDIR)/$(PREFIX)/bin
destrules=$(DESTDIR)/$(UDEVDIR)/rules.d
.PHONY: install
install:
	install -Dm755 tkey-runapp $(destbin)/tkey-runapp
	install -Dm755 hidread $(destbin)/hidread
	install -Dm755 run-tkey-qemu $(destbin)/run-tkey-qemu
	install -Dm644 system/60-tkey.rules $(destrules)/60-tkey.rules
.PHONY: uninstall
uninstall:
	rm -f \
	$(destbin)/tkey-runapp \
	$(destbin)/hidread \
	$(destbin)/run-tkey-qemu \
	$(destrules)/60-tkey.rules \
.PHONY: reload-rules
reload-rules:
	udevadm control --reload
	udevadm trigger

podman:
	podman run --rm --mount type=bind,source=$(CURDIR),target=/src --mount type=bind,source=$(CURDIR)/../tkey-libs,target=/tkey-libs -w /src -it ghcr.io/tillitis/tkey-builder:5rc2 make -j

TKEY_DEVTOOLS_VERSION ?= $(shell git describe --dirty --always | sed -n "s/^v\(.*\)/\1/p")
TKEY_RUNAPP_VERSION ?= $(TKEY_DEVTOOLS_VERSION)
HIDREAD_VERSION ?= $(TKEY_DEVTOOLS_VERSION)

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-runapp
tkey-runapp:
	cd cmd/tkey-runapp && \
	CGO_ENABLED=$(BUILD_CGO_ENABLED) go build -ldflags "-w -X main.version=$(TKEY_RUNAPP_VERSION) -buildid=" -trimpath -buildvcs=false && \
	mv tkey-runapp ../../

.PHONY: hidread
hidread:
	cd cmd/hidread && \
	go build -ldflags "-w -X main.version=$(HIDREAD_VERSION) -buildid=" -trimpath -buildvcs=false && \
	mv hidread ../../

.PHONY: clean
clean:
	rm -f tkey-runapp
	rm -f hidread

.PHONY: lint
lint:
	cd cmd/hidread && golangci-lint run
	cd cmd/tkey-runapp && golangci-lint run

# Extra target just for CI that excludes hidread since we can't build
# CGO dependencies with current tkey-builder.
.PHONY: cilint
cilint:
	cd cmd/tkey-runapp && golangci-lint run
