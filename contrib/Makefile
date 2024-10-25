# Copyright (C) 2022, 2023 - Tillitis AB
# SPDX-License-Identifier: GPL-2.0-only

# image produced by build-image targets
BUILDQEMUIMAGE=qemu-local
BUILDAPPIMAGE=tkey-app-builder-local

# default images used when running a container
APPIMAGE=tkey-app-builder-local
EMUIMAGE=ghcr.io/tillitis/tkey-qemu-tk1-23.03.1

help: all

all:
	@echo "Targets:"
	@echo "run-app               Run a shell using image '$(APPIMAGE)' for app development"
	@echo "pull-app              Pull down the image '$(APPIMAGE)'"
	@echo "build-app-image       Build a development image named '$(BUILDAPPIMAGE)'"
	@echo "                      A newly built image can be used like: make IMAGE=$(BUILDAPPIMAGE) run"


	@echo "run-emu               Run a shell using image '$(EMUIMAGE)' for the TKey emulator"
	@echo "pull-emu              Pull down the image '$(EMUIMAGE)'"
	@echo "build-emu-image       Build a toolchain image named '$(BUILDEMUIMAGE)'"
	@echo "                      A newly built image can be used like: make IMAGE=$(BUILDEMUIMAGE) run"

.PHONY: run-app
run-app:
	podman run --rm --mount type=bind,source="`pwd`/../",target=/build -w /build -it $(BUILDAPPIMAGE)

.PHONY: pull-app
pull-app:
	podman pull $(APPIMAGE)

.PHONY: build-app-image
build-app-image:
	podman build -t $(BUILDAPPIMAGE) -f tkey-app-builder.dockerfile

.PHONY: run-emu
run-emu:
	podman run --rm --mount type=bind,source="`pwd`/../",target=/build -w /build -it $(EMUIMAGE)

.PHONY: pull-emu
pull-emu:
	podman pull $(EMUIMAGE)

.PHONY: build-emu-image
build-emu-image:
	podman build -t $(EMUIMAGE) -f tkey-qemu.dockerfile
