# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`goui` is a Go cross-platform native UI library with a Flutter-style self-drawing engine: it draws every widget itself rather than using OS-native controls. Module path is `github.com/user/goui` (Go 1.24.2). Code comments are in Chinese.

## Build & run prerequisites (read first — non-obvious)

The rendering backend is **Skia via CGO**, not pure OpenGL (the README is out of date — see "Doc drift" below). Almost every package transitively imports `internal/canvas`, which imports `goskia`, so virtually all builds/tests need the full native toolchain:

1. **`CGO_ENABLED=1`** plus a C toolchain (gcc/mingw) on `PATH`. Examples set it explicitly: `$env:CGO_ENABLED='1'`.
2. **Sibling repo `F:\syproject\goskia`** must exist — `go.mod` has `replace github.com/hoonfeng/goskia => F:\syproject\goskia`.
3. **`libSkiaSharp.dll`** (in repo root) is the native Skia library; it must be on the DLL search path at runtime, i.e. in the working directory or next to the `.exe`. This is why prebuilt `.exe`s live in the repo root alongside the DLL.
4. **`fonts/`** must be reachable at runtime (next to the exe or in cwd). `internal/canvas/fontutil.go` loads `fonts/AlibabaPuHuiTi-3-*.ttf` first, then falls back to system fonts. Run from the repo root, or copy `fonts/` next to the exe.
5. **Windows/Win32 is the only working platform.** `x11` (linux) and `cocoa` (darwin) window backends exist behind build tags but are not functional.

## Common commands (PowerShell)

```powershell
$env:CGO_ENABLED='1'   # required for every build/run/test below

# Windowed apps (need libSkiaSharp.dll + fonts/ in cwd → run from repo root)
go run ./cmd/guitest/          # PRIMARY component test app (Web-style widgets + theme + CSS)
go run ./examples/demo/        # declarative JSON-driven + Web-component demo

# Headless render-to-PNG (no window/OpenGL — the practical visual-validation loop)
go run ./examples/skiawidget/  # full Widget→Element→Layout→Paint→Skia pipeline → skiawidget_output.png
go run ./examples/skiapaint/   # raw SkiaCanvas drawing → skiapaint_output.png

go run ./cmd/checkgoskia/      # smoke-test that goskia/Skia is wired (creates a raster surface)

# Build to exe (matches the in-repo workflow; exe must sit beside libSkiaSharp.dll + fonts/)
go build -o demo.exe ./examples/demo/

# Tests (all need CGO + DLL + goskia sibling)
go test ./internal/...
go test ./internal/widget/ -run TestStateButton -v -count=1   # single test
go vet ./...
```

`examples/skiatest/main.go` is tagged `//go:build ignore` and won't build with the others.

## Architecture

Layered, top to bottom: **app → widget → layout → render → canvas → window/platform**, with `event` and `types` cross-cutting.

### Widget / Element separation (Flutter model)
- `internal/widget/widget.go`: `Widget` is immutable config; `Element` is the mutable runtime instance. Each `Widget.CreateElement()` produces its `Element`.
- **Always construct roots/children via `widget.CreateElementFor(w)`, never `w.CreateElement()` directly.** Go struct embedding (e.g. a concrete type embedding `StatefulWidget`/`StatelessWidget`, or `Column`/`Row` embedding `Flex`) makes the promoted `CreateElement()` capture the *embedded* receiver type, losing the outer concrete type and breaking `CreateState()` / element reuse. `CreateElementFor` detects and repairs this — see the long comment in `widget.go`.
- `StatefulWidget` + `State`: `State.SetState()` (via `BaseState.SetState` in `state.go`) calls `element.Rebuild()` and the **global** hook `widget.OnNeedsRepaint`, which `app.Run` wires to `Pipeline.MarkNeedsRepaint()`.

### Repaint vs. relayout (deliberate design — easy to break)
`SetState` and event handling trigger **repaint only, never relayout**. Re-layout calls `buildTree()` which rebuilds the Element tree and would wipe per-Element runtime state (input text, focus, hover, cursor blink). Visual-only changes (focus, hover, cursor) just need a repaint. If a change actually affects *size*, you must call `MarkNeedsLayout()` explicitly. This rationale is documented at length around the `OnNeedsRepaint` wiring in `internal/app/app.go`.

### Layout: BoxConstraints
`internal/layout` — parent passes `BoxConstraints` down, child returns `Size` up. `LayoutContext` / `LayoutResult`. Set `LAYOUT_DEBUG=1` for layout tracing.

### Render pipeline
`internal/render/pipeline.go` owns the root Element and the final Canvas. `PerformLayout()` recursively `Build()`s then `Layout()`s the tree; `Render()` clears → `Paint()`s from root → `Flush()`es; gated by `needsLayout`/`needsRepaint` dirty flags. `EnsureLayout()` runs before event processing so `HitTest(x,y)` has geometry. HitTest walks children in reverse (topmost first) testing `Offset`/`Size` bounds.

### Canvas abstraction
`internal/canvas/canvas.go` defines the `Canvas` interface (`DrawRect/RoundedRect/Circle/Line/Path/Text/Image`, transforms, clip, `Flush`). Implementations: **`SkiaCanvas`** (primary — two modes: **GPU** via goskia `NewGPUSurfaceFromFBO`, rendering Skia directly into the window's GL framebuffer (FBO 0); and **raster** into an `image.RGBA` for headless/PNG via `NewSkiaCanvas`), `softcanvas`, `aacanvas`. (The old `GLCanvas` OpenGL-1.1 self-drawing + texture-upload Canvas was **removed** when SkiaCanvas gained GPU mode; so was the unused `internal/gpu` abstraction.) Drawing style is `paint.Paint` (`internal/paint`: style/color/stroke/opacity/gradients). Geometry/colors are in `internal/types` (Color, Point, Size, Rect, Matrix4, EdgeInsets).

### App framework & main loop
`internal/app/app.go`: `Application.Run(Config)` locks the OS thread (GL requirement), creates a GPU-mode `SkiaCanvas` (`NewSkiaCanvasGPU`, wrapping the window's GL framebuffer), the `Pipeline`, and the root Element. **Display path:** Skia renders **directly** into the window's GL framebuffer (FBO 0) on the GPU; `Pipeline.Render → canvas.Flush → ctx.FlushAndSubmit` submits the work and `SwapBuffers` shows it — no `image.RGBA` readback, no texture upload, no per-frame PNG. (Skia's GPU backend **is** the OpenGL backend, so the GL context — `wglCreateContext`/`MakeCurrent`/`SwapBuffers`/pixel format with stencil — remains the foundation.) GPU resources are explicitly `Release`d (`releaseGPUResources`) after the loop, **before** the window's GL context is destroyed, else goskia finalizers crash on exit (exit 1). The loop: pump platform events → `EnsureLayout` → process queued UI events → render-if-dirty → present. Idle uses blocking `WaitMessage` unless a focused element needs cursor-blink (then a 16 ms tick). Event routing (in `app.go`) is substantial: hover enter/leave, **mouse capture on down** (so up is delivered even outside the element), focus-on-down, drag-threshold detection → `DragStart/Move/End/Drop`, IME candidate-window positioning, and bubbling up the `Parent()` chain. `ShortcutManager` intercepts `KeyDown` before focus routing.

### Window / platform
`internal/window` is an interface; per-OS impls sit behind build tags: `win32` (working; raw `syscall` to user32/gdi32/opengl32/imm32), `x11`, `cocoa`. `internal/platform` is blank-imported (`_ "internal/platform"`) to register the platform, which sets the `window.NewWindow` / `window.SetIMECandidatePos` function vars.

### Declarative / Web-style API (the modern surface)
Beyond the low-level structs, the way apps are actually written (see `cmd/guitest`, `examples/demo`):
- **HTML-like constructors** in `internal/widget/html.go`: `Div(...)`, `H1/H2/P`, etc.
- **CSS-like styling**: `widget.Define(name, Style{...})` style classes + `widget.SetTheme(Theme{...})` design tokens (`style.go`, `theme.go`).
- **JSON-driven UI**: `widget.LoadConfig(data, handlers)` / `LoadConfigFile(path, handlers)` / `BuildFromSpec` (`declarative.go`) — build/modify UI from external JSON without recompiling.

### Validation & visual-test subsystem (a recurring theme here)
- `internal/validate` — a `Suite` runs 3 phases over an Element tree: Build-structure check → Layout-sanity check (no negative/INF/NaN sizes) → per-Element `Validate()` consistency. Register `Scene`s, then `ExecuteAll`.
- `internal/validate/visual` — pixel analysis (e.g. blank-frame detection).
- `internal/capture` — screenshot via OpenGL `glReadPixels`, PNG save, and image `Compare`/`DiffReport`.
- `internal/validation` — a separate validator package.
- The `Verifiable` interface (`Validate() []error`) and `widget.ValidateElementTree` let elements self-check. In practice, headless render-to-PNG (`skiawidget`/`skiapaint`) plus inspecting the output PNGs is the main visual-validation loop.

## Doc drift — verify before trusting

- **README.md describes the old OpenGL self-drawing engine** and example apps `examples/hello` and `examples/todo` that **no longer exist**. The renderer is now Skia (`SkiaCanvas`); OpenGL survives only as the texture-upload display path.
- **`scripts/validate.go`** orchestrates many examples that have been deleted/renamed (`self_validation`, `layout_validation`, `agent`, `validate_api`, `auto_validate`, `visual_validate`, `hello`, `todo`). Don't run it as-is. The current example set is exactly: `cmd/guitest`, `examples/demo`, `examples/skiapaint`, `examples/skiawidget` (+ ignored `skiatest`). Other `scripts/*.go` are standalone `package main` helpers (`go run ./scripts/<file>.go`), several also stale.
- **`svg/`** is a separate, self-contained SVG rendering library with its own README/docs — it is **not** part of the core goui import graph; don't conflate it with the UI engine.
- **`源码备份/`** is a source backup; ignore. `.trae/documents/` holds PRD / architecture planning docs (background only). `.Pair/instructions.md` records the maintainer's preferred communication style.
