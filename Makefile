.PHONY: all
all: tkey-runapp

DESTDIR=/
PREFIX=/usr/local
SYSTEMDDIR=/etc/systemd
UDEVDIR=/etc/udev
destbin=$(DESTDIR)/$(PREFIX)/bin
destman1=$(DESTDIR)/$(PREFIX)/share/man/man1
destunit=$(DESTDIR)/$(SYSTEMDDIR)/user
destrules=$(DESTDIR)/$(UDEVDIR)/rules.d
.PHONY: install
install:
	install -Dm755 tkey-ssh-agent $(destbin)/tkey-ssh-agent
	strip $(destbin)/tkey-ssh-agent
	install -Dm644 system/tkey-ssh-agent.1 $(destman1)/tkey-ssh-agent.1
	gzip -n9f $(destman1)/tkey-ssh-agent.1
	install -Dm644 system/tkey-ssh-agent.service.tmpl $(destunit)/tkey-ssh-agent.service
	sed -i -e "s,##BINDIR##,$(PREFIX)/bin," $(destunit)/tkey-ssh-agent.service
	install -Dm644 system/60-tkey.rules $(destrules)/60-tkey.rules
.PHONY: uninstall
uninstall:
	rm -f \
	$(destbin)/tkey-ssh-agent \
	$(destunit)/tkey-ssh-agent.service \
	$(destrules)/60-tkey.rules \
	$(destman1)/tkey-ssh-agent.1.gz
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
