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

// Interior bounds constants — proportions of the rendered jar image defining
// where the fairy can move (inside the glass walls, below the lid, above the base).
const (
	jarInteriorTop    = 0.2943
	jarInteriorBottom = 0.9119
	jarInteriorLeft   = 0.0693
	jarInteriorRight  = 0.9307
)

// jarRenderedRect computes the position and size of the jar image as actually
// rendered inside the container, accounting for ImageFillContain letterboxing.
// imgAspect is the jar image's width/height ratio.
func jarRenderedRect(containerW, containerH, imgAspect float32) (x, y, w, h float32) {
	containerAspect := containerW / containerH
	if containerAspect > imgAspect {
		// Container wider than jar — pillarboxed (gaps on sides)
		x = (containerW - containerH*imgAspect) / 2
		y = 0
		w = containerH * imgAspect
		h = containerH
	} else {
		// Container taller than jar — letterboxed (gaps top/bottom)
		x = 0
		y = (containerH - containerW/imgAspect) / 2
		w = containerW
		h = containerW / imgAspect
	}
	return x, y, w, h
}

// positionCircle positions and resizes a circle at the fairy's current position,
// mapping normalized 0.0–1.0 coordinates into the jar's interior region.
func (l *fairyJarLayout) positionCircle(circle *canvas.Circle, diameter, containerWidth, containerHeight float32) {
	circle.Resize(fyne.NewSize(diameter, diameter))

	imgAspect := l.fairy.jarBack.Aspect()
	if imgAspect == 0 {
		// No image loaded — fall back to full-container positioning.
		circle.Move(fyne.NewPos(
			float32(l.fairy.posX)*containerWidth-diameter/2,
			float32(l.fairy.posY)*containerHeight-diameter/2,
		))
		return
	}

	jarX, jarY, jarW, jarH := jarRenderedRect(containerWidth, containerHeight, imgAspect)

	posX := float32(l.fairy.posX)
	posY := float32(l.fairy.posY)
	pixelX := jarX + (jarInteriorLeft+posX*(jarInteriorRight-jarInteriorLeft))*jarW - diameter/2
	pixelY := jarY + (jarInteriorTop+posY*(jarInteriorBottom-jarInteriorTop))*jarH - diameter/2

	circle.Move(fyne.NewPos(pixelX, pixelY))
}
