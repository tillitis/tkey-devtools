# Copyright (C) 2024 Tillitis AB
# SPDX-License-Identifier: GPL-2.0-only

FROM alpine:3.20

RUN apk add --no-cache \
    clang \
    clang-extra-tools \
    go \
    lld \
    llvm \
    make
