# Gebaute Weboberfläche

`npm run build` im Ordner `web/` legt die gebaute Oberfläche in `app/` ab.
Der Go-Server bettet sie beim Kompilieren per `go:embed` ein.

`app/` wird nicht eingecheckt. Diese Datei sorgt dafür, dass der Ordner
immer existiert und `go build` auch ohne gebaute Oberfläche funktioniert.
