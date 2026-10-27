# umwand

umwand opens Word, Excel and PowerPoint files as OpenDocument. Register it as
the handler for `.docx`, `.xlsx` and `.pptx`, and double-clicking one converts
it to `.odt`, `.ods` or `.odp` and opens the result in whatever application
handles OpenDocument on the machine. The original is never changed.

It exists for a desktop that has settled on OpenDocument as its working format
but keeps receiving Office files. Converting by hand each time is a chore, and
opening the Office file directly leaves you editing it in its original format,
so the next save writes Office Open XML again. umwand makes the conversion the
act of opening.

## Install

From the Excelano apt repository:

```bash
curl -fsSL https://excelano.com/apt/setup.sh | sudo sh   # one-time
sudo apt install umwand
umwand --register
```

The package installs umwand as a candidate handler. `umwand --register` makes
it the default for your account, and `umwand --unregister` hands the Office
types back to the next application registered for them.

## What happens on open

The converted file is written beside the original under the same name with the
new extension, so `report.docx` produces `report.odt`. An existing file is never
overwritten: a second conversion of a changed original becomes
`report (2).odt`, following the file manager's own numbering. When a conversion
already exists and is newer than the original, umwand opens it rather than
converting again, so reopening the same file does not pile up copies.

A file opened from a temporary location, which is where mail clients and
browsers put attachments, is converted into an `umwand` folder in your Documents
folder instead, because an edit saved next to the attachment would be lost. The
same applies to a file in a folder you cannot write to.

LibreOffice does the conversion when it is installed, through a private profile
so that a LibreOffice window you already have open is not disturbed. Without
it, [this2that](https://github.com/excelano/this2that)'s `docx2odt` and
`xlsx2ods` convert Word and Excel files, and presentations open unconverted.
this2that carries a document's structure rather than its page layout, so
LibreOffice gives the more faithful result.

Success is judged by the file a converter wrote, not by its exit status: the
output has to be an OpenDocument package of the expected type. When a converter
fails, the next one is tried, and when none succeeds the original opens in the
next application registered for its type, with a notification saying why.
Digitally signed files go straight to that application too, since conversion
would invalidate the signature.

A conversion that takes more than a moment shows a notification while it runs,
which disappears when the document opens.

Macro-enabled files (`.docm`, `.xlsm`, `.pptm`) are not handled, and open as
they would without umwand installed.

## Building from source

```bash
go build -o umwand .
go test ./...
packaging/build-deb.sh   # prints the path of the .deb it built
```

`go.mod` names the Go version it needs. The desktop entry the package installs
is `packaging/umwand.desktop`; a build run from the clone needs it copied to
`~/.local/share/applications/` before `--register` will accept it.

## License

MIT
