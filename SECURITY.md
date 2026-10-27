# Security Policy

## Reporting a vulnerability

Please report suspected vulnerabilities privately through GitHub Security Advisories at https://github.com/excelano/umwand/security/advisories/new. If you would rather not use GitHub, email david.anderson@excelano.com instead. I aim to respond within seven days.

Please do not open public issues for security problems.

## Supported versions

Security fixes ship in a new package version. Update with `sudo apt upgrade umwand`. There are no maintained release branches; only the latest version is supported.

## What umwand does

umwand runs locally, only when a file is opened through it. It makes no network connections of its own: no telemetry, no update check and no remote logging. It runs no daemon and keeps no record of the files it opens.

**The input files are the attack surface.** umwand opens the Office package itself only to look for a signature, and hands the conversion to LibreOffice or this2that, so converting a file you received from someone else exposes you to whatever those converters would be exposed to. Keep them current. umwand passes each path to a converter as a separate argument, never through a shell, so a crafted file name cannot become a command.

## What umwand writes

The converted file, beside the original or in the `umwand` folder under Documents, created without overwriting anything that is already there. A LibreOffice profile under `~/.cache/umwand/` used only for conversion. With `--register` and `--unregister`, the entries for the Office types in `~/.config/mimeapps.list`. The original file is never modified, moved or deleted.
