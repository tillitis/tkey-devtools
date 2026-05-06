
# This is the tag in the https://github.com/tillitis/tillitis-key1 repo with
# the firmware that our TKey/QEMU will run. The published tkey-qemu image will
# have this as part of its name, the tag is then used for versioning the image
# (could be updates to the TKey QEMU machine).
ARG TKEYREPO_TAG=TK1-24.03

# This is what we'll actually checkout when building the firmware. It
# should be the release commit.
ARG TKEYREPO_TREEISH=1c90b1aa3dbfb4e62039683ee6049ae8af608498

FROM docker.io/library/ubuntu:24.04 as qemu-builder

RUN apt-get -qq update -y \
    && DEBIAN_FRONTEND=noninteractive \
       apt-get install -y --no-install-recommends \
               build-essential \
               ca-certificates \
               git \
               libglib2.0-dev \
               ninja-build \
               python3 \
               python3-venv

# Cleaning up /usr/local since we will later COPY all from there
RUN rm -rf \
    /usr/local/bin/* \
    /usr/local/repo-commit-*

RUN git clone -b tk1 --depth=1 https://github.com/tillitis/qemu /src/qemu \
    && mkdir /src/qemu/build
WORKDIR /src/qemu/build
RUN ../configure --target-list=riscv32-softmmu --disable-werror \
    && make -j "$(nproc --ignore=2)" \
    && make install \
    && git >/usr/local/repo-commit-tillitis--qemu describe --all --always --long --dirty

FROM ghcr.io/tillitis/tkey-builder:4 AS firmware-builder

ARG TKEYREPO_TREEISH

RUN git clone https://github.com/tillitis/tillitis-key1 /src/tkey
WORKDIR /src/tkey/hw/application_fpga
# QEMU needs the .elf, but we build .bin to check sum
RUN git checkout ${TKEYREPO_TREEISH} \
    && make firmware.bin && sha512sum -c firmware.bin.sha512 \
    && make firmware.elf && cp -af firmware.elf firmware-noconsole.elf \
    && make clean \
    && sed -i "s/-DNOCONSOLE//" Makefile \
    && make firmware.elf && cp -af firmware.elf firmware-console.elf \
    && git >/usr/local/repo-commit-tillitis--key1 describe --all --always --long --dirty


# Our QEMU "runtime" image
FROM docker.io/library/ubuntu:24.04

ARG TKEYREPO_TAG

RUN apt-get -qq update -y \
    && DEBIAN_FRONTEND=noninteractive \
       apt-get install -y --no-install-recommends \
               libglib2.0-0 \
               libusb-1.0-0 \
               libpixman-1-0 \
               python3 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=qemu-builder /usr/local/ /usr/local
COPY --from=qemu-builder /src/qemu/tools/tk1/qemu_usb_mux.py /usr/local/bin/
COPY --from=firmware-builder /usr/local/repo-commit-tillitis--key1 /usr/local/
COPY --from=firmware-builder /src/tkey/hw/application_fpga/firmware-noconsole.elf /tkey-firmware-noconsole.elf
COPY --from=firmware-builder /src/tkey/hw/application_fpga/firmware-console.elf   /tkey-firmware-console.elf

CMD [ "qemu-system-riscv32" \
    , "-nographic" \
    , "-chardev", "serial,id=chrid,path=/pty-on-host" \
    , "-M", "tk1,fifo=chrid" \
    , "-bios", "/tkey-firmware-noconsole.elf" \
    , "-d", "trace:riscv_trap,guest_errors" \
]

LABEL org.opencontainers.image.description="Tillitis TKey QEMU machine with firmware ${TKEYREPO_TAG}"
