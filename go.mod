module github.com/midnightyell/koiImport

go 1.24.1

require gitea.local/smalloy/koiApi v0.0.0

replace gitea.local/smalloy/koiApi => ../koiApi

require golang.org/x/text v0.25.0 // indirect
