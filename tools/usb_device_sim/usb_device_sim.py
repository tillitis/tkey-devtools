#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2025 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

import argparse
import enum
import os
import pty
from select import select
from typing import Optional


class IOEndpoint(enum.IntEnum):
    DEBUG = 0x20
    CDC = 0x40


class Deframer:
    def __init__(self):
        self.endpoint: Optional[int] = None
        self.length: int = 0
        self.cdc_bytes: bytearray = bytearray()
        self.debug_bytes: bytearray = bytearray()

    def put_char(self, char: int):
        if self.endpoint is None:
            self.endpoint = char
            self.length = 0
        elif self.length == 0:
            self.length = char
        else:
            if self.endpoint == IOEndpoint.CDC:
                self.cdc_bytes.append(char)
            elif self.endpoint == IOEndpoint.DEBUG:
                self.debug_bytes.append(char)
            else:
                print(f"Unhandled endpoint: 0x{self.endpoint:02X}")

            self.length -= 1
            if self.length == 0:
                self.endpoint = None


if __name__ == "__main__":
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument("UART_PTS")
    args = arg_parser.parse_args()

    uart_pts_path = args.UART_PTS

    cdc_ptm_fd, cdc_pts_fd = pty.openpty()
    cdc_ptm = os.fdopen(cdc_ptm_fd, "r+b", buffering=0)
    cdc_pts_path = os.readlink(f"/proc/self/fd/{cdc_pts_fd}")
    print(f"Fake CDC device available at:   {cdc_pts_path}")

    debug_ptm_fd, debug_pts_fd = pty.openpty()
    debug_ptm = os.fdopen(debug_ptm_fd, "r+b", buffering=0)
    debug_pts_path = os.readlink(f"/proc/self/fd/{debug_pts_fd}")
    print(f"Fake DEBUG device available at: {debug_pts_path}")

    deframer = Deframer()

    with open(uart_pts_path, "r+b", buffering=0) as uart_pts:
        r_filenos = [uart_pts.fileno(), cdc_ptm.fileno()]

        while True:
            r_event, _, _ = select(r_filenos, [], [])

            if cdc_ptm.fileno() in r_event:
                in_data = cdc_ptm.read(1)
                out_data = b"@\x01" + in_data

                print(f"cdc   -> framer:          {in_data}")
                print(f"         framer -> uart:  {out_data}")
                print("")

                uart_pts.write(out_data)

            if debug_ptm.fileno() in r_event:
                in_data = debug_ptm.read(1)
                out_data = b"@\x01" + in_data

                print(f"debug -> framer:          {in_data}")
                print(f"         framer -> uart:  {out_data}")
                print("")

                uart_pts.write(out_data)

            if uart_pts.fileno() in r_event:
                in_data = uart_pts.read(1)
                print(f"         framer <- uart:  {in_data}")

                deframer.put_char(in_data[0])

                if len(deframer.cdc_bytes) > 0:
                    out_data = bytes(deframer.cdc_bytes)
                    deframer.cdc_bytes = bytearray()

                    print(f"cdc   <- framer:          {out_data}")

                    cdc_ptm.write(out_data)

                if len(deframer.debug_bytes) > 0:
                    out_data = bytes(deframer.debug_bytes)
                    deframer.debug_bytes = bytearray()

                    print(f"debug <- framer:          {out_data}")

                    debug_ptm.write(out_data)
                print("")
