# Release notes

## v0.0.2

- Update tkeyclient version because of a vulnerability leaving some
  USSs unused. Keys might have changed since earlier versions! Read
  more here:

  https://github.com/tillitis/tkeyclient/security/advisories/GHSA-4w7r-3222-8h6v

- Add a new option flag to `tkey-runapp`: `--force-full-uss` to force
  full use of the 32 byte USS digest.
