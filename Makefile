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
	strip $(destbin)/tkey-runapp
	install -Dm755 run-tkey-qemu $(destbin)/run-tkey-qemu
	install -Dm644 system/60-tkey.rules $(destrules)/60-tkey.rules
.PHONY: uninstall
uninstall:
	rm -f \
	$(destbin)/tkey-runapp \
	$(destbin)/run-tkey-qemu \
	$(destrules)/60-tkey.rules \
.PHONY: reload-rules
reload-rules:
	udevadm control --reload
	udevadm trigger

podman:
	podman run --rm --mount type=bind,source=$(CURDIR),target=/src --mount type=bind,source=$(CURDIR)/../tkey-libs,target=/tkey-libs -w /src -it ghcr.io/tillitis/tkey-builder:2 make -j

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-runapp
tkey-runapp:
	go build ./cmd/tkey-runapp

.PHONY: clean
clean:
	rm -f tkey-runapp

.PHONY: lint
lint:
	$(MAKE) -C gotools
	GOOS=linux   ./gotools/golangci-lint run
	GOOS=windows ./gotools/golangci-lint run
