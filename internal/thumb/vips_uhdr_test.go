package thumb

import (
	"slices"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVipsHasGainmap(t *testing.T) {
	VipsInit()

	t.Run("Nil", func(t *testing.T) {
		assert.False(t, vipsHasGainmap(nil))
	})

	t.Run("NoGainmap", func(t *testing.T) {
		img, err := vips.LoadImageFromFile("testdata/example.jpg", vips.NewImportParams())
		require.NoError(t, err)
		defer img.Close()

		assert.False(t, vipsHasGainmap(img))
	})

	t.Run("GainmapData", func(t *testing.T) {
		img, err := vips.LoadImageFromFile("testdata/example.jpg", vips.NewImportParams())
		require.NoError(t, err)
		defer img.Close()
		img.SetBlob("gainmap-data", []byte("gainmap"))

		assert.True(t, vipsHasGainmap(img))
	})
}

func TestVipsRemoveThumbnailMetadata(t *testing.T) {
	VipsInit()

	t.Run("PreservesUltraHDRMetadata", func(t *testing.T) {
		img, err := vips.LoadImageFromFile("testdata/example.jpg", vips.NewImportParams())
		require.NoError(t, err)
		defer img.Close()

		img.SetBlob("gainmap-data", []byte("gainmap"))
		img.SetBlob("xmp-data", []byte("xmp"))
		img.SetDouble("gainmap-max-content-boost", 4)
		img.SetDouble("gainmap-min-content-boost", 1)

		require.NoError(t, vipsRemoveThumbnailMetadata(img, true))

		fields := img.GetFields()
		assert.True(t, slices.Contains(fields, "gainmap-data"))
		assert.True(t, slices.Contains(fields, "xmp-data"))
		assert.True(t, slices.Contains(fields, "gainmap-max-content-boost"))
		assert.True(t, slices.Contains(fields, "gainmap-min-content-boost"))
		assert.Equal(t, []byte("gainmap"), img.GetBlob("gainmap-data"))
		assert.Equal(t, []byte("xmp"), img.GetBlob("xmp-data"))
	})

	t.Run("StripsMetadataForRegularImage", func(t *testing.T) {
		img, err := vips.LoadImageFromFile("testdata/example.jpg", vips.NewImportParams())
		require.NoError(t, err)
		defer img.Close()

		img.SetBlob("xmp-data", []byte("xmp"))
		require.NoError(t, vipsRemoveThumbnailMetadata(img, false))

		assert.NotContains(t, img.GetFields(), "xmp-data")
	})
}
