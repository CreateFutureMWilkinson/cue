package fairy

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// fairyJarLayout is a custom Fyne layout that sizes jar layers to fill the
// container and sizes fairy circles proportionally to the container width.
type fairyJarLayout struct {
	fairy *FairyCharacter
}

func (l *fairyJarLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 200)
}

func (l *fairyJarLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	w := size.Width
	h := size.Height

	// Position jar PNGs to fill the entire container.
	l.positionJarLayers(size)

	// Position fairy circles (body + glow layers).
	l.positionFairyCircles(w, h)

	// Position hidden indicator (maintained for compatibility).
	l.fairy.indicator.Resize(fyne.NewSize(fairyIndicatorSize, fairyIndicatorSize))
}

// positionJarLayers positions the jar back and front PNG layers.
func (l *fairyJarLayout) positionJarLayers(size fyne.Size) {
	origin := fyne.NewPos(0, 0)

	l.fairy.jarBack.Resize(size)
	l.fairy.jarBack.Move(origin)
	l.fairy.jarBack.Refresh()

	l.fairy.jarFront.Resize(size)
	l.fairy.jarFront.Move(origin)
	l.fairy.jarFront.Refresh()

	l.fairy.layoutRefreshCount++
}

// positionFairyCircles positions the body circle and glow layers.
func (l *fairyJarLayout) positionFairyCircles(containerWidth, containerHeight float32) {
	bodyDiam := containerWidth * bodyRatio
	l.positionCircle(l.fairy.bodyCircle, bodyDiam, containerWidth, containerHeight)

	glowDiam := containerWidth * glowRatio
	for i, gl := range l.fairy.glowLayers {
		t := float32(i+1) / float32(fairyGlowLayerCount)
		d := bodyDiam + (glowDiam-bodyDiam)*t
		l.positionCircle(gl, d, containerWidth, containerHeight)
	}
}

// jarRenderedRect computes the position and size of the jar image as actually
// rendered inside the container, accounting for ImageFillContain letterboxing.
// imgAspect is the jar image's width/height ratio.
func jarRenderedRect(containerW, containerH, imgAspect float32) (x, y, w, h float32) {
	return 0, 0, 0, 0
}

// positionCircle positions and resizes a circle at the fairy's current position.
func (l *fairyJarLayout) positionCircle(circle *canvas.Circle, diameter, containerWidth, containerHeight float32) {
	circle.Resize(fyne.NewSize(diameter, diameter))
	circle.Move(fyne.NewPos(
		float32(l.fairy.posX)*containerWidth-diameter/2,
		float32(l.fairy.posY)*containerHeight-diameter/2,
	))
}
