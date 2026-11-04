package gallows

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

// everything is every stroke of a finished drawing at the largest budget the
// parts list can serve, which is the widest the picture ever gets.
func everything() []Stroke {
	return append(FrameStrokes(), FigureStrokes(len(Parts), len(Parts), 1)...)
}

func abs(f float32) float32 { return float32(math.Abs(float64(f))) }

func drawnFor(budget, from, to int) []int {
	var out []int
	for w := from; w <= to; w++ {
		out = append(out, PartsDrawn(budget, w))
	}
	return out
}

func TestPartCountNeverDropsBelowTheClassicSix(t *testing.T) {
	for budget := 0; budget <= CoreParts; budget++ {
		if got := PartCount(budget); got != CoreParts {
			t.Errorf("budget %d: %d", budget, got)
		}
	}
}

func TestPartCountFollowsABiggerBudgetUpToThePartsThereAre(t *testing.T) {
	for _, c := range []struct{ budget, want int }{{7, 7}, {len(Parts), len(Parts)}, {len(Parts) + 5, len(Parts)}} {
		if got := PartCount(c.budget); got != c.want {
			t.Errorf("budget %d: %d, want %d", c.budget, got, c.want)
		}
	}
}

func TestTheClassicBudgetDrawsOnePartPerWrongGuess(t *testing.T) {
	for wrong := 0; wrong <= CoreParts; wrong++ {
		if got := PartsDrawn(CoreParts, wrong); got != wrong {
			t.Errorf("wrong %d: %d", wrong, got)
		}
	}
}

func TestNothingIsDrawnBeforeTheFirstWrongGuess(t *testing.T) {
	for budget := 0; budget <= 12; budget++ {
		if got := PartsDrawn(budget, 0); got != 0 {
			t.Errorf("budget %d: %d", budget, got)
		}
	}
}

func TestTheFigureIsAlwaysFinishedOnTheLastGuess(t *testing.T) {
	for budget := 1; budget <= 12; budget++ {
		if got := PartsDrawn(budget, budget); got != PartCount(budget) {
			t.Errorf("budget %d: %d, want %d", budget, got, PartCount(budget))
		}
	}
}

func TestTheDrawingNeverGoesBackwards(t *testing.T) {
	for budget := 1; budget <= 12; budget++ {
		for wrong := 1; wrong <= budget; wrong++ {
			if PartsDrawn(budget, wrong) < PartsDrawn(budget, wrong-1) {
				t.Errorf("budget %d, wrong %d", budget, wrong)
			}
		}
	}
}

func TestATightBudgetDrawsMoreThanOnePartPerGuess(t *testing.T) {
	if got := drawnFor(4, 1, 4); !slices.Equal(got, []int{2, 3, 5, 6}) {
		t.Errorf("got %v", got)
	}
}

func TestAGenerousBudgetReachesForTheDetailParts(t *testing.T) {
	if got := drawnFor(8, 1, 8); !slices.Equal(got, []int{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("got %v", got)
	}
}

func TestMoreGuessesThanPartsRepeatsAStageRatherThanOverrunning(t *testing.T) {
	drawn := drawnFor(12, 1, 12)
	if drawn[len(drawn)-1] != len(Parts) {
		t.Errorf("last stage %d", drawn[len(drawn)-1])
	}
	repeats := false
	for i := 1; i < len(drawn); i++ {
		repeats = repeats || drawn[i-1] == drawn[i]
	}
	if !repeats {
		t.Errorf("no stage repeats: %v", drawn)
	}
	if slices.Max(drawn) > len(Parts) {
		t.Errorf("overran the parts: %v", drawn)
	}
}

func TestABudgetOfNothingCannotDivideByZero(t *testing.T) {
	if PartsDrawn(0, 0) != 0 || PartsDrawn(0, 1) != CoreParts {
		t.Errorf("got %d and %d", PartsDrawn(0, 0), PartsDrawn(0, 1))
	}
}

func TestExtraWrongGuessesAreClampedToTheBudget(t *testing.T) {
	if PartsDrawn(6, 99) != PartsDrawn(6, 6) {
		t.Errorf("got %d", PartsDrawn(6, 99))
	}
}

func TestASettledFigureIsOneStrokePerPart(t *testing.T) {
	if n := len(FigureStrokes(6, 0, 1)); n != 0 {
		t.Errorf("0 wrong: %d strokes", n)
	}
	for _, wrong := range []int{3, 5} {
		if n := len(FigureStrokes(6, wrong, 1)); n != wrong {
			t.Errorf("%d wrong: %d strokes", wrong, n)
		}
	}
}

func TestTheNewestPartIsMissingAtTheStartOfItsAnimation(t *testing.T) {
	if n := len(FigureStrokes(6, 3, 0)); n != 2 {
		t.Errorf("progress 0: %d strokes", n)
	}
	if n := len(FigureStrokes(6, 3, 0.5)); n != 3 {
		t.Errorf("progress 0.5: %d strokes", n)
	}
}

func TestTheFaceArrivesOnlyWithTheFinishedFigure(t *testing.T) {
	for _, s := range FigureStrokes(6, 5, 1) {
		if s.Ink == Face {
			t.Error("the face arrived early")
		}
	}
	if n := len(FigureStrokes(6, 6, 1)); n != CoreParts+len(FaceStrokes()) {
		t.Errorf("finished figure has %d strokes", n)
	}
}

func TestTheFaceIsDrawnOnWithTheStrokeThatFinishesTheFigure(t *testing.T) {
	mid := FigureStrokes(6, 6, 0.5)
	if len(mid) != CoreParts+len(FaceStrokes()) {
		t.Fatalf("%d strokes", len(mid))
	}
	face := FaceStrokes()
	if mouth := mid[len(mid)-1]; mouth.Length() >= face[len(face)-1].Length() {
		t.Errorf("the mouth is already whole at %v", mouth.Length())
	}
}

func TestAHalfDrawnLineStopsAtItsMidpoint(t *testing.T) {
	half := line(Point{0, 0}, Point{10, 0}, 1, Figure).Partial(0.5)
	if want := []Point{{0, 0}, {5, 0}}; !reflect.DeepEqual(half.Points, want) {
		t.Errorf("points %v", half.Points)
	}
}

func TestPartialAtTheEndsIsAllOrNothing(t *testing.T) {
	head := Head.Stroke()
	for _, tt := range []float32{0, -1} {
		if p := head.Partial(tt); len(p.Points) != 0 {
			t.Errorf("Partial(%v) has %d points", tt, len(p.Points))
		}
	}
	for _, tt := range []float32{1, 2} {
		if p := head.Partial(tt); !reflect.DeepEqual(p, head) {
			t.Errorf("Partial(%v) changed the stroke", tt)
		}
	}
}

func TestAHalfDrawnHeadIsAnOpenArcOfAboutHalfTheCircle(t *testing.T) {
	head := Head.Stroke()
	if !head.Closed {
		t.Fatal("the head is not closed")
	}
	half := head.Partial(0.5)
	if half.Closed {
		t.Error("a half-drawn head is closed")
	}
	if r := half.Length() / head.Length(); abs(r-0.5) >= 0.01 {
		t.Errorf("half is %v of the head", r)
	}
}

// Flattened into straight segments it is a hair short of a true circle.
func TestAClosedHeadMeasuresTheWholeWayRound(t *testing.T) {
	circumference := float32(2 * math.Pi * headRadius)
	if l := Head.Stroke().Length(); abs(l-circumference) >= circumference*0.01 {
		t.Errorf("length %v, circumference %v", l, circumference)
	}
}

func TestEveryStrokeHasSomewhereToGo(t *testing.T) {
	for _, s := range everything() {
		if len(s.Points) < 2 || s.Length() <= 0 || s.Width <= 0 {
			t.Errorf("stroke %+v", s)
		}
	}
}

func TestTheWholeDrawingStaysInsideTheDesignBox(t *testing.T) {
	for _, s := range everything() {
		m := s.Width / 2
		for _, p := range s.Points {
			if p.X-m < 0 || p.X+m > DesignWidth || p.Y-m < 0 || p.Y+m > DesignHeight {
				t.Errorf("%v leaves the design box", p)
			}
		}
	}
}

func TestTheDrawingIsRoughlyCenteredInItsBox(t *testing.T) {
	left, right := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for _, s := range everything() {
		for _, p := range s.Points {
			left, right = min(left, p.X), max(right, p.X)
		}
	}
	if c := (left + right) / 2; abs(c-DesignWidth/2) >= 5 {
		t.Errorf("centered on %v", c)
	}
}

func TestTheBodyHangsOffTheRopeWithoutAGap(t *testing.T) {
	frame := FrameStrokes()
	rope := frame[len(frame)-1]
	if rope.Ink != Rope {
		t.Fatalf("the last frame stroke is %v, not the rope", rope.Ink)
	}
	ropeEnd := rope.Points[len(rope.Points)-1]
	if d := ropeEnd.Distance(Head.Stroke().Points[0]); d >= 0.01 {
		t.Errorf("a %v gap between rope and head", d)
	}
}

func TestTheLimbsStartOnTheBody(t *testing.T) {
	torso := Torso.Stroke()
	chin, hip := torso.Points[0], torso.Points[1]
	if chin.Distance(Point{headX, chinY}) >= 0.01 {
		t.Errorf("chin %v", chin)
	}
	for _, arm := range []Part{LeftArm, RightArm} {
		shoulder := arm.Stroke().Points[0]
		if shoulder.X != chin.X || shoulder.Y <= chin.Y || shoulder.Y >= hip.Y {
			t.Errorf("shoulder %v", shoulder)
		}
	}
	for _, leg := range []Part{LeftLeg, RightLeg} {
		if start := leg.Stroke().Points[0]; start != hip {
			t.Errorf("leg starts at %v, hip is %v", start, hip)
		}
	}
}

func TestTheDetailPartsHangOffTheLimbsTheyFinish(t *testing.T) {
	for _, pair := range [][2]Part{{LeftArm, LeftHand}, {RightArm, RightHand}, {LeftLeg, LeftFoot}, {RightLeg, RightFoot}} {
		limb := pair[0].Stroke().Points
		if start := pair[1].Stroke().Points[0]; start != limb[len(limb)-1] {
			t.Errorf("part %d starts at %v, limb ends at %v", pair[1], start, limb[len(limb)-1])
		}
	}
}

func TestTheFaceFitsInsideTheHead(t *testing.T) {
	head := Point{headX, headY}
	for _, s := range FaceStrokes() {
		for _, p := range s.Points {
			if p.Distance(head) >= headRadius {
				t.Errorf("%v is outside the head", p)
			}
		}
	}
}

// reviewBox is the box the canvas gets inside the stage panel from the review
// screenshot, and minimumWindowBox the same in the shortest window the app
// opens, where the panel's fixed width leaves only the height to shrink.
var (
	reviewBox        = [2]float32{300, 560}
	minimumWindowBox = [2]float32{300, 430}
)

// coverage is how much of the box's width and height the drawing covers.
func coverage(width, height float32) (float32, float32) {
	f := Fit(width, height)
	return f.DrawnWidth() / width, f.DrawnHeight() / height
}

func TestEveryStrokeStaysInsideTheInkBox(t *testing.T) {
	for budget := 0; budget <= len(Parts)+4; budget++ {
		for wrong := 0; wrong <= budget+1; wrong++ {
			strokes := append(FrameStrokes(), FigureStrokes(budget, wrong, 1)...)
			strokes = append(strokes, FaceStrokes()...)
			for _, s := range strokes {
				e := s.Width / 2
				for _, p := range s.Points {
					if p.X-e < InkLeft || p.X+e > InkRight || p.Y-e < InkTop || p.Y+e > InkBottom {
						t.Errorf("budget %d, wrong %d: %v escapes the ink box", budget, wrong, p)
					}
				}
			}
		}
	}
}

func TestTheReviewPanelGetsADrawingThatFillsIt(t *testing.T) {
	width, height := reviewBox[0], reviewBox[1]
	f := Fit(width, height)
	// 240.5 design units of ink across a 300px panel, less a 4% margin either
	// side.
	if abs(f.Scale-1.1476) >= 0.001 {
		t.Errorf("scale %v", f.Scale)
	}
	if abs(f.DrawnWidth()-276) >= 0.01 {
		t.Errorf("drawn %v wide", f.DrawnWidth())
	}
	if abs(f.DrawnHeight()-338.5) >= 0.1 {
		t.Errorf("drawn %v tall", f.DrawnHeight())
	}
	across, down := coverage(width, height)
	if across <= 0.9 {
		t.Errorf("only %v of the width", across)
	}
	if down >= across {
		t.Error("the height should not be the constraint")
	}
}

func TestTheDrawingLandsAMarginInFromEachEdge(t *testing.T) {
	width, height := reviewBox[0], reviewBox[1]
	f := Fit(width, height)
	left := f.Map(Point{InkLeft, InkTop})
	right := f.Map(Point{InkRight, InkBottom})
	if abs(left.X-width*margin) >= 0.01 {
		t.Errorf("left edge %v", left)
	}
	if abs(right.X-width*(1-margin)) >= 0.01 {
		t.Errorf("right edge %v", right)
	}
	// Centered vertically, with the same slack above as below.
	if abs((height-right.Y)-left.Y) >= 0.01 {
		t.Errorf("%v to %v in %v", left, right, height)
	}
}

func TestTheShortestWindowStillFillsItsPanel(t *testing.T) {
	width, height := minimumWindowBox[0], minimumWindowBox[1]
	across, down := coverage(width, height)
	if max(across, down) <= 0.9 {
		t.Errorf("%v across and %v down is not filling anything", across, down)
	}
	if across > 1 || down > 1 {
		t.Error("the drawing overflows its box")
	}
	if Fit(width, height).Scale <= 0.9 {
		t.Error("the drawing shrank away")
	}
}

func TestAVeryTallPanelStopsGrowingTheDrawingAtTheWidth(t *testing.T) {
	width := reviewBox[0]
	settled := Fit(width, 560).Scale
	for _, height := range []float32{1000, 4000, 20000} {
		if got := Fit(width, height).Scale; got != settled {
			t.Errorf("at %v tall the scale is %v, want %v", height, got, settled)
		}
	}
}

func TestTheDrawingKeepsItsShapeWhateverTheBox(t *testing.T) {
	shape := float32(InkWidth / InkHeight)
	for _, box := range [][2]float32{{300, 560}, {300, 120}, {900, 300}, {80, 900}} {
		width, height := box[0], box[1]
		f := Fit(width, height)
		if drawn := f.DrawnWidth() / f.DrawnHeight(); abs(drawn-shape) >= 0.001 {
			t.Errorf("%vx%v: shape %v", width, height, drawn)
		}
		if f.DrawnWidth() > width+0.01 {
			t.Errorf("%vx%v too wide", width, height)
		}
		if f.DrawnHeight() > height+0.01 {
			t.Errorf("%vx%v too tall", width, height)
		}
	}
}

func TestTheLinesGetThickerAsTheDrawingGetsBigger(t *testing.T) {
	width, height := reviewBox[0], reviewBox[1]
	big := Fit(width, height)
	small := Fit(width/4, height/4)
	if w := big.LineWidth(frameWidth); w <= 7 {
		t.Errorf("frame is %vpx", w)
	}
	if r := big.LineWidth(frameWidth) / small.LineWidth(frameWidth); abs(r-4) >= 0.001 {
		t.Errorf("widths do not track the scale: ratio %v", r)
	}
}

func TestAShrunkenDrawingKeepsHairlineStrokesVisible(t *testing.T) {
	f := Fit(InkWidth/10, InkHeight/10)
	if f.Scale >= 0.1 {
		t.Errorf("scale %v", f.Scale)
	}
	if w := f.LineWidth(faceWidth); w != 1 {
		t.Errorf("face line %vpx", w)
	}
}

func TestAnEmptyBoxAsksForNothingImpossible(t *testing.T) {
	f := Fit(0, 0)
	if f.Scale != 0 || f.DrawnWidth() != 0 || f.LineWidth(frameWidth) != 1 {
		t.Errorf("fit %+v, line %v", f, f.LineWidth(frameWidth))
	}
}

// A rough stroke is the same every time for one seed, so it cannot shimmer
// while it is drawn on, and it stays near the line it roughens.
func TestARoughStrokeIsSteadyAndStaysNearItsLine(t *testing.T) {
	line := Stroke{Points: []Point{{0, 0}, {100, 0}}, Width: 6}
	a, b := line.Rough(7, 1.5), line.Rough(7, 1.5)
	if len(a.Points) < 20 || len(a.Points) != len(b.Points) {
		t.Fatalf("points %d and %d", len(a.Points), len(b.Points))
	}
	for i := range a.Points {
		if a.Points[i] != b.Points[i] {
			t.Fatal("the same seed drew two different strokes")
		}
		if y := a.Points[i].Y; y > 1.5 || y < -1.5 {
			t.Errorf("point %d strays %v from the line", i, y)
		}
	}
	if a.Points[0].X >= 0 || a.Points[len(a.Points)-1].X <= 100 {
		t.Error("a rough stroke should run a little past both ends")
	}
	if c := line.Rough(8, 1.5); c.Points[5] == a.Points[5] {
		t.Error("another seed drew the same wobble")
	}
}

// A head is many short segments, each shorter than the rough step, and a
// rough head must still go all the way round.
func TestARoughHeadGoesAllTheWayRound(t *testing.T) {
	head := Head.Stroke()
	rough := head.Rough(3, 1.4)
	if len(rough.Points) < len(head.Points) {
		t.Fatalf("a head of %d points roughened into %d", len(head.Points), len(rough.Points))
	}
	if got, want := rough.Length(), head.Length(); got < want {
		t.Errorf("rough head is %v long, the head %v", got, want)
	}
}
