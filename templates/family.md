# {{ .Fam.Name }}

* First seen: {{ .Fam.FirstSeen }}
* Aliases: {{- $aliases := .Fam.Aliases}}
{{- range .Fam.Aliases }}
 {{- . }}
{{- if ne . ($aliases | last) }}, {{- end }}
{{- end }}
* Samples:
{{- range .Fam.Samples }}
    - {{ .SHA256 }} | {{ .Platform}} | {{ .Category}} | {{ .FileFormat }}
{{- end}}

{{- $files := .Files}}
{{- range .Fam.Samples }}

{{- $file := index $files .SHA256}}

##  {{ .Description }}
### Basic Properties

| Property | Value |
| --- | --- |
| Size | {{ $file.Size }} bytes |
| CRC32 | {{ $file.Crc32 }} |
| MD5 | {{ $file.MD5 }} |
| SHA1 | {{ $file.SHA1 }} |
| SHA256 | {{ $file.SHA256 }} |
| SHA512 | {{ $file.SHA512 }} |
| Ssdeep | {{ $file.Ssdeep }} |
| Magic | {{ $file.Magic }}  |
| Packer | {{ range $file.Packer }}{{ . }}<br />{{ end }} |
| TrID | {{ range $file.TriD }}{{ . }}<br />{{ end }} |

## Antivirus Scan

```diff
{{- range $k, $v := $file.MultiAV.last_scan }}
{{- if $v.output }}
- {{ $k | title }}: {{ $v.output }}
{{- else }}
+ {{ $k | title }}: clean
{{- end }}
{{- end }}
```

{{- end }}

## References

{{- range .Fam.References }}
- [{{ . }}]({{ . }})
{{- end }}
