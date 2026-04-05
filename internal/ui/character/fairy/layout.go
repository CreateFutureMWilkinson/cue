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
// All circles share the same center (computed from the body diameter) so that
// glow layers remain concentric with the body.
func (l *fairyJarLayout) positionFairyCircles(containerWidth, containerHeight float32) {
	jarAspect := l.fairy.jarBack.Aspect()
	var jarRect jarRenderInfo
	if jarAspect != 0 {
		jarRect.x, jarRect.y, jarRect.width, jarRect.height = jarRenderedRect(containerWidth, containerHeight, jarAspect)
	}

	bodyDiameter := containerWidth * bodyRatio

	// Compute the body center once — all circles will share this center.
	centerX, centerY := l.circleCenter(bodyDiameter, jarRect, containerWidth, containerHeight)

	l.positionCircleAtCenter(l.fairy.bodyCircle, bodyDiameter, centerX, centerY)

	glowDiameter := containerWidth * glowRatio
	for i, glowLayer := range l.fairy.glowLayers {
		interpolation := float32(i+1) / float32(fairyGlowLayerCount)
		diameter := bodyDiameter + (glowDiameter-bodyDiameter)*interpolation
		l.positionCircleAtCenter(glowLayer, diameter, centerX, centerY)
	}
}

// jarRenderInfo holds the rendered jar rectangle information.
type jarRenderInfo struct {
	x, y, width, height float32
}

// circleCenter computes the center point for a circle of the given diameter
// at the fairy's current position, mapping normalized 0.0–1.0 coordinates
// into the jar's interior region (or full container if no jar is loaded).
func (l *fairyJarLayout) circleCenter(diameter float32, jar jarRenderInfo, containerWidth, containerHeight float32) (float32, float32) {
	fairyPosX := float32(l.fairy.posX)
	fairyPosY := float32(l.fairy.posY)

	// No jar loaded - use full container positioning
	if jar.width == 0 || jar.height == 0 {
		return fairyPosX * containerWidth, fairyPosY * containerHeight
	}

	// Calculate jar interior bounds in pixel coordinates
	interiorLeft := jar.x + jarInteriorLeft*jar.width
	interiorRight := jar.x + jarInteriorRight*jar.width
	interiorTop := jar.y + jarInteriorTop*jar.height
	interiorBottom := jar.y + jarInteriorBottom*jar.height

	// Map fairy position to interior space, ensuring entire circle stays inside
	availableWidth := interiorRight - interiorLeft - diameter
	availableHeight := interiorBottom - interiorTop - diameter

	// Center = top-left + diameter/2
	centerX := interiorLeft + fairyPosX*availableWidth + diameter/2
	centerY := interiorTop + fairyPosY*availableHeight + diameter/2
	return centerX, centerY
}

// positionCircleAtCenter resizes a circle and positions it so its center
// is at the given (centerX, centerY) point.
func (l *fairyJarLayout) positionCircleAtCenter(circle *canvas.Circle, diameter, centerX, centerY float32) {
	circle.Resize(fyne.NewSize(diameter, diameter))
	circle.Move(fyne.NewPos(centerX-diameter/2, centerY-diameter/2))
}
