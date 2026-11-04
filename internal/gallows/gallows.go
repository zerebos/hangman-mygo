// Package gallows is the gallows drawing as plain geometry, with no UI types,
// so the whole picture can be checked by go test.
//
// The picture is described once in a fixed DesignWidth × DesignHeight box,
// and Fit maps it onto whatever rectangle the window gives it. What Fit sizes
// to the rectangle is the ink box rather than the design box, so the panel is
// filled by the drawing and not by the empty margin the coordinates carry.
//
// A drawing is a list of strokes, and a stroke is a polyline, even the head,
// which is a circle flattened into one. That one shape is what makes the
// draw-on animation simple: Stroke.Partial cuts a polyline at a fraction of
// its length, so a limb grows out from the shoulder and the head sweeps round
// from the rope.
package gallows

import "math"

// DesignWidth and DesignHeight are the box every coordinate is in.
const (
	DesignWidth  = 300.0
	DesignHeight = 350.0
)

// How many straight segments a full circle and the mouth arc are flattened
// into.
const (
	circleSegments = 48
	arcSegments    = 12
)

// The frame: what is on screen before the first wrong guess.
const (
	groundY     = 322.0
	groundX0    = 34.0
	groundX1    = 138.0
	postX       = 86.0
	beamY       = 34.0
	beamX       = 214.0
	bracePostY  = 90.0
	braceBeamX  = 142.0
	ropeBottomY = 74.0
)

// The victim.
const (
	headX      = beamX
	headRadius = 28.0
	headY      = ropeBottomY + headRadius
	chinY      = headY + headRadius
	hipY       = 210.0
	shoulderY  = 152.0
	handY      = 196.0
	armReach   = 42.0
	footY      = 268.0
	legReach   = 38.0
	// The ticks that finish the limbs when the budget needs more than six
	// parts.
	handReach = 12.0
	handDrop  = 8.0
	footReach = 16.0
	footDrop  = 2.0
)

// The face, drawn on with the last part.
const (
	eyeY        = 94.0
	eyeOffset   = 9.0
	eyeArm      = 5.0
	mouthY      = 120.0
	mouthRadius = 9.0
	mouthFrom   = 200.0
	mouthTo     = 340.0
)

// Line widths in design units.
const (
	frameWidth  = 7.0
	ropeWidth   = 4.0
	figureWidth = 6.0
	faceWidth   = 3.5
)

// The ink box is the rectangle the ink covers inside the design box, half of
// every stroke's width included. The design box has slack in it (nothing is
// drawn in its right-hand fifth), so Fit fits this box instead.
const (
	InkLeft   = groundX0 - frameWidth/2
	InkRight  = headX + armReach + handReach + figureWidth/2
	InkTop    = beamY - frameWidth/2
	InkBottom = groundY + frameWidth/2
	InkWidth  = InkRight - InkLeft
	InkHeight = InkBottom - InkTop
)

// margin is how much of each side of the panel Fit leaves empty, as a
// fraction of that side.
const margin = 0.04

// Point is a point in the design box. Y grows downwards, as on screen.
type Point struct {
	X, Y float32
}

// Distance is how far p is from q.
func (p Point) Distance(q Point) float32 {
	return float32(math.Hypot(float64(q.X-p.X), float64(q.Y-p.Y)))
}

// Lerp is the point t of the way from p to q.
func (p Point) Lerp(q Point, t float32) Point {
	return Point{p.X + (q.X-p.X)*t, p.Y + (q.Y-p.Y)*t}
}

// Ink is the role a stroke plays, so the UI can color it from the theme.
type Ink int

const (
	// Frame is the gallows: base, post, beam and brace.
	Frame Ink = iota
	// Rope is the rope hanging off the beam.
	Rope
	// Figure is the victim's head, torso and limbs.
	Figure
	// Face is drawn only once the figure is finished.
	Face
)

// Stroke is one continuous line of the drawing.
type Stroke struct {
	// Points is the polyline in design units. Fewer than two draw nothing.
	Points []Point
	// Closed joins the last point back to the first. Only the finished head
	// is closed.
	Closed bool
	// Width is the line's thickness in design units.
	Width float32
	Ink   Ink
}

func line(from, to Point, width float32, ink Ink) Stroke {
	return Stroke{Points: []Point{from, to}, Width: width, Ink: ink}
}

// Length is the total length of the polyline, its closing segment included.
func (s Stroke) Length() float32 {
	var total float32
	for i := 1; i < len(s.Points); i++ {
		total += s.Points[i-1].Distance(s.Points[i])
	}
	if s.Closed && len(s.Points) > 2 {
		total += s.Points[len(s.Points)-1].Distance(s.Points[0])
	}
	return total
}

// Partial is the first t of the stroke by length, which is what it looks like
// part way through being drawn. t <= 0 has no points and t >= 1 is the stroke
// unchanged.
func (s Stroke) Partial(t float32) Stroke {
	if t >= 1 {
		return s
	}
	cut := Stroke{Width: s.Width, Ink: s.Ink}
	if t <= 0 || len(s.Points) < 2 {
		return cut
	}

	target := s.Length() * t
	cut.Points = []Point{s.Points[0]}
	var walked float32
	// The closing segment is walked like any other, so a head cut at 0.99
	// stops just short of its start rather than snapping shut.
	segments := len(s.Points) - 1
	if s.Closed && len(s.Points) > 2 {
		segments++
	}
	for i := range segments {
		from := s.Points[i]
		to := s.Points[(i+1)%len(s.Points)]
		step := from.Distance(to)
		if walked+step >= target {
			var along float32
			if step > 0 {
				along = (target - walked) / step
			}
			cut.Points = append(cut.Points, from.Lerp(to, along))
			break
		}
		walked += step
		cut.Points = append(cut.Points, to)
	}
	return cut
}

// Part is one part of the victim, in the order the parts are drawn. The first
// six are the classic hangman; the four ticks after them appear only when the
// budget needs more than six stages.
type Part int

const (
	Head Part = iota
	Torso
	LeftArm
	RightArm
	LeftLeg
	RightLeg
	LeftHand
	RightHand
	LeftFoot
	RightFoot
)

// Parts is every part, in drawing order.
var Parts = [...]Part{Head, Torso, LeftArm, RightArm, LeftLeg, RightLeg, LeftHand, RightHand, LeftFoot, RightFoot}

// CoreParts is the classic six, the fewest that still make a whole hangman.
const CoreParts = 6

// Stroke is where the part is and how thick it is drawn.
func (p Part) Stroke() Stroke {
	shoulder := Point{headX, shoulderY}
	hip := Point{headX, hipY}
	leftHand := Point{headX - armReach, handY}
	rightHand := Point{headX + armReach, handY}
	leftFoot := Point{headX - legReach, footY}
	rightFoot := Point{headX + legReach, footY}

	switch p {
	case Head:
		return Stroke{Points: circle(Point{headX, headY}, headRadius), Closed: true, Width: figureWidth, Ink: Figure}
	case Torso:
		return line(Point{headX, chinY}, hip, figureWidth, Figure)
	case LeftArm:
		return line(shoulder, leftHand, figureWidth, Figure)
	case RightArm:
		return line(shoulder, rightHand, figureWidth, Figure)
	case LeftLeg:
		return line(hip, leftFoot, figureWidth, Figure)
	case RightLeg:
		return line(hip, rightFoot, figureWidth, Figure)
	case LeftHand:
		return line(leftHand, Point{leftHand.X - handReach, handY + handDrop}, figureWidth, Figure)
	case RightHand:
		return line(rightHand, Point{rightHand.X + handReach, handY + handDrop}, figureWidth, Figure)
	case LeftFoot:
		return line(leftFoot, Point{leftFoot.X - footReach, footY + footDrop}, figureWidth, Figure)
	default:
		return line(rightFoot, Point{rightFoot.X + footReach, footY + footDrop}, figureWidth, Figure)
	}
}

// FrameStrokes is the gallows, there from the first frame.
func FrameStrokes() []Stroke {
	return []Stroke{
		line(Point{groundX0, groundY}, Point{groundX1, groundY}, frameWidth, Frame),
		line(Point{postX, groundY}, Point{postX, beamY}, frameWidth, Frame),
		line(Point{postX, beamY}, Point{beamX, beamY}, frameWidth, Frame),
		line(Point{postX, bracePostY}, Point{braceBeamX, beamY}, frameWidth, Frame),
		line(Point{beamX, beamY}, Point{beamX, ropeBottomY}, ropeWidth, Rope),
	}
}

// FaceStrokes is two crossed eyes and a frown, drawn only on a finished
// figure.
func FaceStrokes() []Stroke {
	strokes := make([]Stroke, 0, 5)
	for _, eye := range []float32{headX - eyeOffset, headX + eyeOffset} {
		strokes = append(strokes,
			line(Point{eye - eyeArm, eyeY - eyeArm}, Point{eye + eyeArm, eyeY + eyeArm}, faceWidth, Face),
			line(Point{eye + eyeArm, eyeY - eyeArm}, Point{eye - eyeArm, eyeY + eyeArm}, faceWidth, Face),
		)
	}
	return append(strokes, Stroke{
		Points: arc(Point{headX, mouthY}, mouthRadius, mouthFrom, mouthTo, arcSegments),
		Width:  faceWidth,
		Ink:    Face,
	})
}

// PartCount is how many parts the figure has for a budget: never fewer than
// the classic six and never more than Parts holds.
func PartCount(budget int) int {
	return min(max(budget, CoreParts), len(Parts))
}

// PartsDrawn is how much of the figure wrong guesses out of budget have
// earned. The figure is always complete on the last guess: a tight budget
// draws more than one part a guess, and a generous one repeats a stage near
// the end rather than inventing body parts.
func PartsDrawn(budget, wrong int) int {
	total := PartCount(budget)
	if budget <= 0 {
		if wrong == 0 {
			return 0
		}
		return total
	}
	n := min(max(wrong, 0), budget) * total
	return (n + budget - 1) / budget
}

// FigureStrokes is the victim after wrong guesses out of budget, with the parts the
// latest guess earned drawn only progress of the way. A progress of 1 is the
// settled picture.
func FigureStrokes(budget, wrong int, progress float32) []Stroke {
	drawn := PartsDrawn(budget, wrong)
	settled := PartsDrawn(budget, max(wrong-1, 0))
	progress = min(max(progress, 0), 1)

	strokes := make([]Stroke, 0, drawn+5)
	push := func(s Stroke, whole bool) {
		if whole {
			strokes = append(strokes, s)
		} else if cut := s.Partial(progress); len(cut.Points) >= 2 {
			strokes = append(strokes, cut)
		}
	}
	for i, part := range Parts[:drawn] {
		push(part.Stroke(), i < settled)
	}
	// The face is the full stop, so it arrives with the stroke that finishes
	// the figure rather than as a stage of its own.
	if drawn > 0 && drawn == PartCount(budget) {
		for _, s := range FaceStrokes() {
			push(s, settled == drawn)
		}
	}
	return strokes
}

// Fitting is how the design box maps onto the rectangle the drawing was given.
type Fitting struct {
	// Scale is how many real pixels one design unit is worth.
	Scale float32
	// Offset is where the design box's origin lands inside the rectangle.
	// The ink box is what is centered, so this is usually a little outside.
	Offset Point
}

// Map is where p lands, relative to the rectangle's top left.
func (f Fitting) Map(p Point) Point {
	return Point{f.Offset.X + p.X*f.Scale, f.Offset.Y + p.Y*f.Scale}
}

// LineWidth is a design-unit width in real pixels, never thinner than a
// hairline.
func (f Fitting) LineWidth(width float32) float32 {
	return max(width*f.Scale, 1)
}

// DrawnWidth and DrawnHeight are the size of the finished drawing in real
// pixels.
func (f Fitting) DrawnWidth() float32  { return InkWidth * f.Scale }
func (f Fitting) DrawnHeight() float32 { return InkHeight * f.Scale }

// Fit fits the drawing into a width × height rectangle: as large as it goes
// without distorting it or crossing the margin, and centered in what is left.
func Fit(width, height float32) Fitting {
	usable := float32(1 - 2*margin)
	scale := max(min(width*usable/InkWidth, height*usable/InkHeight), 0)
	return Fitting{
		Scale: scale,
		Offset: Point{
			(width-InkWidth*scale)/2 - InkLeft*scale,
			(height-InkHeight*scale)/2 - InkTop*scale,
		},
	}
}

// circle is a circle flattened into a polyline, starting at the top and going
// clockwise, so the head draws away from the rope.
func circle(center Point, radius float32) []Point {
	points := make([]Point, circleSegments)
	for i := range points {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/circleSegments
		points[i] = Point{center.X + radius*float32(math.Cos(angle)), center.Y + radius*float32(math.Sin(angle))}
	}
	return points
}

// arc is part of a circle, flattened the same way. Angles are in degrees,
// clockwise from three o'clock.
func arc(center Point, radius, from, to float32, segments int) []Point {
	segments = max(segments, 1)
	points := make([]Point, segments+1)
	for i := range points {
		deg := from + (to-from)*float32(i)/float32(segments)
		angle := float64(deg) * math.Pi / 180
		points[i] = Point{center.X + radius*float32(math.Cos(angle)), center.Y + radius*float32(math.Sin(angle))}
	}
	return points
}
