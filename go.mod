module github.com/hoonfeng/paircode

go 1.26

require (
	github.com/go-rod/rod v0.116.2
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/yalue/onnxruntime_go v1.31.0
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/sys v0.44.0
	modernc.org/sqlite v1.53.0
	wb-ui v0.0.0
	wb-ui.com/goja v0.0.0-00010101000000-000000000000 // indirect
)

require github.com/dop251/goja v0.0.0-20260719185829-0fc1d42c1dc9

require (
	github.com/dlclark/regexp2/v2 v2.5.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20260707082822-2a407d02d01a // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hoonfeng/goskia v0.0.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/ysmood/fetchup v0.2.3 // indirect
	github.com/ysmood/goob v0.4.0 // indirect
	github.com/ysmood/got v0.40.0 // indirect
	github.com/ysmood/gson v0.7.3 // indirect
	github.com/ysmood/leakless v0.9.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	modernc.org/libc v1.73.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace (
	github.com/hoonfeng/goskia => ../goskia
	wb-ui => ../wb-ui
	wb-ui.com/goja => ../wb-ui/goja
)
