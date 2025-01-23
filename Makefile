.PHONY: all
all: tkey-runapp hidread

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
	podman run --rm --mount type=bind,source=$(CURDIR),target=/src --mount type=bind,source=$(CURDIR)/../tkey-libs,target=/tkey-libs -w /src -it ghcr.io/tillitis/tkey-builder:2 make -j

TKEY_DEVTOOLS_VERSION ?= $(shell git describe --dirty --always | sed -n "s/^v\(.*\)/\1/p")
TKEY_RUNAPP_VERSION ?= $(TKEY_DEVTOOLS_VERSION)
HIDREAD_VERSION ?= $(TKEY_DEVTOOLS_VERSION)

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-runapp
tkey-runapp:
	cd cmd/tkey-runapp && \
	go build -ldflags "-w -X main.version=$(TKEY_RUNAPP_VERSION) -buildid=" -trimpath && \
	mv tkey-runapp ../../

.PHONY: hidread
hidread:
	cd cmd/hidread && \
	go build -ldflags "-w -X main.version=$(HIDREAD_VERSION) -buildid=" -trimpath && \
	mv hidread ../../

.PHONY: clean
clean:
	rm -f tkey-runapp
	rm -f hidread

.PHONY: lint
lint:
	$(MAKE) -C gotools
	GOOS=linux   ./gotools/golangci-lint run
	GOOS=windows ./gotools/golangci-lint run
