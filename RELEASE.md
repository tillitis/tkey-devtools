# Release notes

## v0.0.3

- Update tkeyclient to v1.3.1 to handle TKey Unlocked (product ID 8)
  as a Bellatrix when it comes to USS digest handling.

- Only allow `--force-full-uss` when either `--uss` or `--uss-file` is
  used.

Full
[changelog](https://github.com/tillitis/tkey-devtools/compare/v0.0.2...v0.0.3).

## v0.0.2

- Update tkeyclient version because of a vulnerability leaving some
  USSs unused. Keys might have changed since earlier versions! Read
  more here:

  https://github.com/tillitis/tkeyclient/security/advisories/GHSA-4w7r-3222-8h6v

- Add a new option flag to `tkey-runapp`: `--force-full-uss` to force
  full use of the 32 byte USS digest.
