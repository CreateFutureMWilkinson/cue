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

// positionFairyCircles positions the body circle and glow layers within the
// jar's interior region, computing the jar rendered rect once for all circles.
func (l *fairyJarLayout) positionFairyCircles(containerWidth, containerHeight float32) {
	jarAspect := l.fairy.jarBack.Aspect()
	var jarRect jarRenderInfo
	if jarAspect != 0 {
		jarRect.x, jarRect.y, jarRect.width, jarRect.height = jarRenderedRect(containerWidth, containerHeight, jarAspect)
	}

	bodyDiameter := containerWidth * bodyRatio
	l.positionCircle(l.fairy.bodyCircle, bodyDiameter, jarRect, containerWidth, containerHeight)

	glowDiameter := containerWidth * glowRatio
	for i, glowLayer := range l.fairy.glowLayers {
		interpolation := float32(i+1) / float32(fairyGlowLayerCount)
		diameter := bodyDiameter + (glowDiameter-bodyDiameter)*interpolation
		l.positionCircle(glowLayer, diameter, jarRect, containerWidth, containerHeight)
	}
}

// jarRenderInfo holds the rendered jar rectangle information.
type jarRenderInfo struct {
	x, y, width, height float32
}

// positionCircle positions and resizes a circle at the fairy's current position,
// mapping normalized 0.0–1.0 coordinates into the jar's interior region.
func (l *fairyJarLayout) positionCircle(circle *canvas.Circle, diameter float32, jar jarRenderInfo, containerWidth, containerHeight float32) {
	circle.Resize(fyne.NewSize(diameter, diameter))

	fairyPosX := float32(l.fairy.posX)
	fairyPosY := float32(l.fairy.posY)

	// No jar loaded - use full container positioning
	if jar.width == 0 || jar.height == 0 {
		circle.Move(fyne.NewPos(
			fairyPosX*containerWidth-diameter/2,
			fairyPosY*containerHeight-diameter/2,
		))
		return
	}

	// Calculate jar interior bounds in pixel coordinates
	interiorLeft := jar.x + jarInteriorLeft*jar.width
	interiorRight := jar.x + jarInteriorRight*jar.width
	interiorTop := jar.y + jarInteriorTop*jar.height
	interiorBottom := jar.y + jarInteriorBottom*jar.height

	// Map fairy position to interior space, ensuring entire circle stays inside
	availableWidth := interiorRight - interiorLeft - diameter
	availableHeight := interiorBottom - interiorTop - diameter

	pixelX := interiorLeft + fairyPosX*availableWidth
	pixelY := interiorTop + fairyPosY*availableHeight
	circle.Move(fyne.NewPos(pixelX, pixelY))
}
