## PhotoPrism — Thumbnails Package

**Last Updated:** November 22, 2026

### Overview

`internal/thumb` builds thumbnails with libvips, handling resize/crop options, color management, metadata stripping, and format export (JPEG/PNG). It also contains bounded helper paths for reading and processing image sources.

### Eager & On-Demand Sizes

`Names` lists sizes generated during indexing. It includes `tile_224` for the UI and local S2 center input. `left_224` and `right_224` remain registered in `Sizes` and render on demand, but are not part of the default eager generation set.

`OpenInputSource` prepares oriented model sources with the configured renderer. Native inputs stay lazy until the selected region is cropped and resized; only the final input pixels are copied into Go memory.

### Constraints

- Uses libvips via govips; initialization is centralized in `VipsInit`.
- Standard thumbnail generation requires libvips 8.14+; Ultra HDR JPEG gain-map resizing and preservation requires libvips 8.18+ with Ultra HDR support enabled.
- Works on files or in-memory buffers; writes outputs with `fs.ModeFile`.
- Generic Go `image.Decode()` / `image.DecodeConfig()` dispatch is intentionally not used for TIFF inputs.
- TIFF originals should be opened through libvips where possible; helper decode paths validate the TIFF header and first IFD offset before calling the direct TIFF decoder.
- ICC handling: if a JPEG lacks an embedded profile but sets EXIF `InteroperabilityIndex` (`R03`/Adobe RGB, `R98`/sRGB, `THM`/thumbnail), we embed an Adobe-compatible profile; otherwise we leave color handling to the embedded profile or sRGB default.
- Ordinary metadata is removed from thumbnails. When an Ultra HDR JPEG is emitted, the gain map, HDRGM parameters, and XMP are retained so libvips can write an Ultra HDR JPEG.

### Goals

- Produce consistent thumbnails for all configured sizes and resample modes.
- Preserve color fidelity when cameras signal color space through EXIF interop tags.
- Keep error paths non-fatal: invalid sizes, missing files, or absent profiles should return errors (not panics). The libvips entry points `Vips` and `vipsConvert` reduce the binding's error to a message while preserving wrapped errors.

### Non-Goals

- Serving or caching thumbnails (handled elsewhere).
- Full ICC workflow management; thumbnail export retains its profile policy while model inputs request sRGB pixels.

### Package Layout (Code Map)

- `vips.go` — main `Vips` entry: load, resize/crop, conditionally preserve Ultra HDR metadata, export; `vipsErr` error reduction.
- `vips_uhdr.go` — gain-map detection and metadata retention for Ultra HDR JPEG thumbnails.
- `verify.go` — `Verify` decode probe; returns the libvips error unreduced.
- `vips_convert.go` — HEIC/HEIF/AVIF and other format conversion via libvips (`vipsConvert`).
- `vips_icc.go` — EXIF InteroperabilityIndex handling and ICC embedding.
- `icc.go` — lists bundled ICC filenames (`IccProfiles`) and `GetIccProfile` helper.
- `open_input.go` — lazy oriented model-input sources, bounded region resampling, and source cleanup.
- `open.go`, `open_jpeg.go` — bounded file decode helpers for non-libvips paths, including explicit JPEG color handling and TIFF-safe dispatch via `pkg/fs`.
- `resample.go`, `sizes.go`, `size.go` — resample options, the predefined size catalog (`MaxSize`, `MaxRenderSize`, `InvalidSize`), and the per-size helpers (`Uncached`, `ExceedsLimit`, `Clamp`, and `Resize`).
- `config.go` — package defaults for `SizeCached`, `SizeOnDemand` and `SizeFace`, assigned from the configuration by `Config.Propagate`.
- `thumb.go` and helpers — naming, caching, file info.
- Tests live alongside sources (`*_test.go`, fixtures under `testdata/`).

### ICC & Interop Handling

- EXIF `InteroperabilityIndex` codes we honor (per EXIF TagNames and regex.info):
  - `R03` → Adobe RGB (1998) compatible (`a98.icc`, etc.)
  - `R98` → sRGB (assumed default; no embed)
  - `THM` → Thumbnail (treated as sRGB; no embed)
- If an ICC profile already exists, we skip embedding.
- Exiftool name differs (libvips expects `exif-ifd4-InteroperabilityIndex`):
  ```
  exiftool -InteropIndex -InteropVersion -icc_profile:all -G -s file.jpg
  ```
- Test Files:
  - `testdata/interop_index.jpg` — R03 interop tag, no ICC (expects Adobe profile embed).
  - `testdata/interop_index_srgb_icc.jpg` — R03 tag with embedded ICC (must remain unchanged).
  - `testdata/interop_index_r98.jpg` — R98 interop tag, no ICC (should stay sRGB without embedding).
  - `testdata/interop_index_thm.jpg` — THM interop tag, no ICC (thumbnail; should remain unchanged).
- References:
  - [EXIF TagNames (InteroperabilityIndex)](https://unpkg.com/exiftool-vendored.pl@10.50.0/bin/html/TagNames/EXIF.html)
  - [Digital-Image Color Spaces: Recommendations and Links](https://regex.info/blog/photo-tech/color-spaces-page7)

### Tests

- Fast scoped: `go test ./internal/thumb -run 'Icc|Vips' -count=1`
- Full: `go test ./internal/thumb -count=1`

### Lint & Formatting

- Format: `make fmt-go`
- Lint: `make lint-go` or `golangci-lint run ./internal/thumb/...`

### Notes

- When adding ICC files, place them in `assets/profiles/icc/` and append to `IccProfiles`.
- Comments for exported identifiers must start with the identifier name (Go style).

### Format Handling

- Cached thumbnails are written as JPEG or PNG and may be reopened through bounded helper paths for crop, preview, and AI follow-up work.
- TIFF is intentionally excluded from generic Go decoder registration in this package and related callers so future code paths cannot reach the unsafe generic TIFF dispatch by accident.
- Already-decoded `image.Image` values are resized, cropped, rotated, and saved through stdlib plus `golang.org/x/image/draw` helpers rather than through a third-party imaging library.
- `Save` writes JPEG/PNG data through `fs.OpenStageFile` and renames the completed sibling into place. Creation permissions follow `fs.ModeFile` filtered by the process umask; the writer closes and syncs before rename.
- HEIC/HEIF and AVIF originals are converted through `vipsConvert`, which uses libvips (via libheif) to load and export the image. Because libheif always applies ISOBMFF `irot`/`imir` container transforms during decode, this conversion path skips the EXIF orientation when libvips reports a `heifload` loader.

### Go 1.26 JPEG Notes

Go `1.26.0` replaced the standard `image/jpeg` encoder and decoder. This package primarily relies on libvips for thumbnail generation, but helper paths and tests that decode or encode JPEGs through Go may produce slightly different pixels or file sizes across toolchain versions.

Observed impact during internal comparison runs (Go `1.25.4` vs `1.26.0`):

- **Compatibility** — No decode failures for 55/55 JPEG fixtures in `assets/samples` on either version.
- **Decoded Pixels** — All scanned JPEG fixtures produced different decoded pixel hashes across versions, even though dimensions were unchanged.
- **Re-Encode Size** — Re-encoded JPEG sizes changed slightly in both directions; aggregate deltas were small (about `+0.014%` at default quality, about `+0.017%` at quality 95 in our fixture set).
- **Throughput** — Micro-benchmark runs showed modest improvements in decode and decode+encode throughput in Go `1.26.0`.

Testing guidance:

- Do not rely on bit-for-bit JPEG output across Go toolchain upgrades.
- Prefer assertions on image dimensions, error-free processing, and perceptual/tolerance metrics where appropriate.
