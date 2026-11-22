package thumb

import (
	"github.com/davidbyttow/govips/v2/vips"
)

// vipsHasGainmap reports whether a libvips image carries Ultra HDR gain-map data.
func vipsHasGainmap(img *vips.ImageRef) bool {
	if img == nil {
		return false
	}

	for _, field := range img.GetFields() {
		if field == "gainmap-data" || field == "gainmap" {
			return true
		}
	}

	return false
}

// vipsRemoveMetadataPreservingGainmap removes ordinary metadata while keeping Ultra HDR data for JPEG export.
func vipsRemoveMetadataPreservingGainmap(img *vips.ImageRef) error {
	return img.RemoveMetadata(
		"gainmap-data",
		"gainmap",
		"gainmap-max-content-boost",
		"gainmap-min-content-boost",
		"gainmap-gamma",
		"gainmap-offset-sdr",
		"gainmap-offset-hdr",
		"gainmap-hdr-capacity-min",
		"gainmap-hdr-capacity-max",
		"gainmap-use-base-cg",
		"gainmap-scale-factor",
		"xmp-data",
	)
}
