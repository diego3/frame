package scene

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"goengine/frameengine/object"
)

// drawParallaxLayers draws every active, non-prototype GameObject with a ParallaxLayer component
// directly onto screen -- not the level-sized worldBuffer Draw blits everything else through --
// so each layer can scroll at its own fraction of the camera's movement (see
// object.ParallaxLayer.ScrollFactor) instead of the uniform 1:1 scroll every world-buffer object
// gets. Drawn in GameObject list order (the same "declare background objects first" convention
// level1.yaml already uses for draw order) and always before the world buffer, so gameplay renders
// on top of every layer regardless of ScrollFactor.
func (m *WorldScene) drawParallaxLayers(screen *ebiten.Image) {
	if m.world == nil {
		return
	}
	var camX, camY float64
	if m.cam != nil {
		camX, camY = m.cam.X, m.cam.Y
	}
	viewportW := float64(screen.Bounds().Dx())
	for _, go_ := range m.world.Objects() {
		if !go_.Active || go_.IsPrototype {
			continue
		}
		layer, ok := go_.GetComponent("parallax_layer").(*object.ParallaxLayer)
		if !ok || layer.Image == nil {
			continue
		}
		t := go_.Transform()
		if t == nil {
			continue
		}
		drawParallaxLayer(screen, layer, t, camX, camY, viewportW)
	}
}

// drawParallaxLayer draws one layer's image at its transform position offset by the camera's
// scroll scaled by the layer's own ScrollFactor (0 = fixed to the camera/infinitely distant, 1 =
// scrolls 1:1 with gameplay), scaled by the transform's ScaleX/ScaleY like Sprite.Draw -- source
// background art (e.g. a 288x176 sky tile) is typically much smaller than the level and needs
// scaling up to cover it, same as any other art in this engine. When Repeat is set, the (scaled)
// image -- typically a narrow strip meant to tile, e.g. a corridor wall panel, rather than stretch
// across the level -- is drawn as many times as needed to fully cover the viewport width.
func drawParallaxLayer(screen *ebiten.Image, layer *object.ParallaxLayer, t *object.Transform, camX, camY, viewportW float64) {
	baseX := t.X - camX*layer.ScrollFactor
	baseY := t.Y - camY*layer.ScrollFactor
	if !layer.Repeat {
		drawImageAt(screen, layer.Image, baseX, baseY, t.ScaleX, t.ScaleY)
		return
	}
	iw := float64(layer.Image.Bounds().Dx()) * t.ScaleX
	for _, x := range parallaxTileXs(baseX, iw, viewportW) {
		drawImageAt(screen, layer.Image, x, baseY, t.ScaleX, t.ScaleY)
	}
}

// parallaxTileXs returns the x positions at which a Repeat layer's image (width iw) must be drawn
// to fully cover the viewport [0, viewportW), aligned to a multiple of iw from baseX so tiles never
// visibly seam or jump as the camera scrolls. Pulled out as a pure function so the tiling math is
// unit-testable without an *ebiten.Image.
func parallaxTileXs(baseX, iw, viewportW float64) []float64 {
	if iw <= 0 {
		return nil
	}
	// Shift baseX into (-iw, 0] so the first tile in the loop below is the leftmost one that can
	// still overlap the viewport, regardless of how far baseX itself has drifted off-screen.
	x := math.Mod(baseX, iw)
	if x > 0 {
		x -= iw
	}
	var xs []float64
	for ; x < viewportW; x += iw {
		xs = append(xs, x)
	}
	return xs
}

func drawImageAt(screen, img *ebiten.Image, x, y, scaleX, scaleY float64) {
	op := &ebiten.DrawImageOptions{}
	// Scale first, then Translate -- same reasoning as Sprite.Draw: GeoM.Scale scales the entire
	// current matrix including translation, so scaling after translating would also move the
	// image's position, not just resize it in place before it's placed at x, y.
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}
