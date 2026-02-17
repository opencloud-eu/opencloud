package imaging

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"time"

	"github.com/kovidgoyal/imaging/apng"
	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/gifmeta"
	"github.com/kovidgoyal/imaging/webp"
)

var _ = fmt.Print

type Frame struct {
	Number      uint          // a 1-based frame number
	TopLeft     image.Point   // location of top-left of this frame w.r.t top left of first frame
	Image       image.Image   `json:"-"` // the actual pixel data
	Delay       time.Duration // the time for which this frame should be visible
	ComposeOnto uint          // the frame number of the frame this frame should be composed onto. 0 means compose onto blank
	Replace     bool          // Do a simple pixel replacement rather than a full alpha blend when compositing this frame
}

type Image struct {
	Frames       []*Frame    // the actual frames of image data. The first frame is guaranteed to be the size of the image.
	Metadata     *meta.Data  // image metadata
	LoopCount    uint        // 0 means loop forever, 1 means loop once, ...
	DefaultImage image.Image `json:"-"` // a "default image" for an animation that is not part of the actual animation
}

func (self *Image) populate_from_apng(p *apng.APNG) {
	self.LoopCount = p.LoopCount
	prev_disposal := apng.DISPOSE_OP_BACKGROUND
	var prev_compose_onto uint
	for _, f := range p.Frames {
		if f.IsDefault {
			self.DefaultImage = f.Image
			continue
		}
		frame := Frame{Number: uint(len(self.Frames) + 1), Image: NormalizeOrigin(f.Image),
			TopLeft: image.Point{X: f.XOffset, Y: f.YOffset},
			Replace: f.BlendOp == apng.BLEND_OP_SOURCE,
			Delay:   time.Duration(float64(time.Second) * f.GetDelay())}
		switch prev_disposal {
		case apng.DISPOSE_OP_NONE:
			frame.ComposeOnto = frame.Number - 1
		case apng.DISPOSE_OP_PREVIOUS:
			frame.ComposeOnto = uint(prev_compose_onto)
		}
		prev_disposal, prev_compose_onto = int(f.DisposeOp), frame.ComposeOnto
		self.Frames = append(self.Frames, &frame)
	}
}

func (f *Frame) Bounds() image.Rectangle {
	return f.Image.Bounds().Add(f.TopLeft)
}

func (f *Frame) ColorModel() color.Model {
	return f.Image.ColorModel()
}

func (f *Frame) At(x, y int) color.Color {
	return f.Image.At(x-f.TopLeft.X, y-f.TopLeft.Y)
}

func (f *Frame) Dx() int { return f.Image.Bounds().Dx() }
func (f *Frame) Dy() int { return f.Image.Bounds().Dy() }

type canvas_t = image.NRGBA

var new_canvas = image.NewNRGBA

func (self *Image) populate_from_webp(p *webp.AnimatedWEBP) {
	// See https://developers.google.com/speed/webp/docs/riff_container#animation
	self.LoopCount = uint(p.Header.LoopCount)
	bgcol := p.Header.BackgroundColor
	// For some reason web viewers treat bgcol as full transparent. Sigh.
	bgcol = image.Transparent
	bg := image.NewUniform(bgcol)
	_, _, _, a := bg.RGBA()
	bg_is_fully_transparent := a == 0
	w, h := int(self.Metadata.PixelWidth), int(self.Metadata.PixelHeight)
	var dispose_prev bool
	for i, f := range p.Frames {
		frame := Frame{
			Number: uint(len(self.Frames) + 1), Image: NormalizeOrigin(f.Frame),
			TopLeft: image.Point{X: 2 * int(f.Header.FrameX), Y: 2 * int(f.Header.FrameY)},
			Replace: !f.Header.AlphaBlend,
			Delay:   time.Millisecond * time.Duration(f.Header.FrameDuration),
		}
		// we want the first frame to have the same size as the canvas, which
		// is not always true in WebP
		if i == 0 && (frame.Dx() < w || frame.Dy() < h || frame.TopLeft != image.Point{}) {
			img := new_canvas(image.Rect(0, 0, w, h))
			dest := image.Rectangle{frame.TopLeft, frame.TopLeft.Add(image.Pt(frame.Bounds().Dx(), frame.Bounds().Dy()))}
			if !bg_is_fully_transparent {
				draw.Draw(img, img.Bounds(), bg, image.Point{}, draw.Src)
				draw.Draw(img, dest, frame.Image, image.Point{}, draw.Over)
			} else {
				draw.Draw(img, dest, frame.Image, image.Point{}, draw.Src)
			}
			frame.Image = img
			frame.TopLeft = image.Point{}
		}

		frame.ComposeOnto = frame.Number - 1
		if dispose_prev {
			// According to the spec dispose only affects the area of the
			// frame, filling it with the background color on disposal, so
			// add an extra frame that clears the prev frame's region and then
			// draw the current frame as gapless frame.
			prev_frame := self.Frames[len(self.Frames)-1]
			b := prev_frame.Image.Bounds()
			if bg_is_fully_transparent && (prev_frame.TopLeft == image.Point{}) && b.Dx() >= w && b.Dy() >= h {
				// prev frame covered entire canvas and background is clear so
				// just clear canvas
				frame.ComposeOnto = 0
			} else {
				img := image.NewNRGBA(b)
				draw.Draw(img, b, bg, image.Point{}, draw.Src)
				if b == frame.Image.Bounds() && prev_frame.TopLeft == frame.TopLeft {
					// prev frame and this frame overlap exactly, so just compose
					// directly without needing an extra frame
					draw.Draw(img, b, frame.Image, image.Point{}, draw.Over)
					frame.Replace = true
					frame.Image = img
				} else {
					// insert gapless frame to dispose previous frame
					nf := Frame{
						Number: frame.Number, Image: img, TopLeft: prev_frame.TopLeft, Replace: true,
						ComposeOnto: prev_frame.Number,
					}
					self.Frames = append(self.Frames, &nf)
					frame.Number++
					frame.ComposeOnto = nf.Number
				}
			}
		}

		dispose_prev = f.Header.DisposalBitSet
		self.Frames = append(self.Frames, &frame)
	}
}

func (self *Image) populate_from_gif(g *gif.GIF) {
	min_gap := gifmeta.CalcMinimumGap(g.Delay)
	prev_disposal := uint8(gif.DisposalBackground)
	var prev_compose_onto uint
	for i, img := range g.Image {
		b := img.Bounds()
		frame := Frame{
			Number: uint(len(self.Frames) + 1), Image: NormalizeOrigin(img), TopLeft: b.Min,
			Delay: gifmeta.CalculateFrameDelay(g.Delay[i], min_gap),
		}
		switch prev_disposal {
		case gif.DisposalNone, 0: // 1
			frame.ComposeOnto = frame.Number - 1
		case gif.DisposalPrevious: // 3
			frame.ComposeOnto = prev_compose_onto
		case gif.DisposalBackground: // 2
			if i > 0 && g.Delay[i-1] == 0 {
				// this is in contravention of the GIF spec but browsers and
				// gif2apng both do this, so follow them. Test images for this
				// are apple.gif and disposal-background-with-delay.gif
				frame.ComposeOnto = frame.Number - 1
			} else {
				// delay present, frame visible, so clear to background as the spec requires
				frame.ComposeOnto = 0
			}
		}
		prev_disposal, prev_compose_onto = g.Disposal[i], frame.ComposeOnto
		self.Frames = append(self.Frames, &frame)
	}
	switch {
	case g.LoopCount == 0:
		self.LoopCount = 0
	case g.LoopCount < 0:
		self.LoopCount = 1
	default:
		self.LoopCount = uint(g.LoopCount) + 1
	}
}

// Create a clone of this image, all data is copied
func (self *Image) Clone() *Image {
	ans := *self
	if ans.DefaultImage != nil {
		ans.DefaultImage = ClonePreservingType(ans.DefaultImage)
	}
	if ans.Metadata != nil {
		ans.Metadata = ans.Metadata.Clone()
	}
	ans.Frames = make([]*Frame, len(self.Frames))
	for i, f := range self.Frames {
		nf := *f
		nf.Image = ClonePreservingType(f.Image)
		ans.Frames[i] = &nf
	}
	return &ans
}

// Coalesce all animation frames so that each frame is a snapshot of the
// animation at that instant.
func (self *Image) Coalesce() {
	if len(self.Frames) == 1 {
		return
	}
	canvas_rect := self.Frames[0].Bounds()
	var canvas *canvas_t
	for i, f := range self.Frames {
		if i == 0 || f.ComposeOnto == 0 {
			canvas = new_canvas(canvas_rect)
		} else {
			canvas = ClonePreservingType(self.Frames[f.ComposeOnto-1].Image).(*canvas_t)
		}
		op := draw.Over
		if f.Replace {
			op = draw.Src
		}
		draw.Draw(canvas, f.Bounds(), f.Image, image.Point{}, op)
		f.Image = canvas
		f.TopLeft = image.Point{}
		f.ComposeOnto = 0
		f.Replace = true
	}
}

// converts a time.Duration to a numerator and denominator of type uint16.
// It finds the best rational approximation of the duration in seconds.
func as_fraction(d time.Duration) (num, den uint16) {
	if d <= 0 {
		return 0, 1
	}

	// Convert duration to seconds as a float64
	val := d.Seconds()

	// Use continued fractions to find the best rational approximation.
	// We look for the convergent that is closest to the original value
	// while keeping the numerator and denominator within uint16 bounds.

	bestNum, bestDen := uint16(0), uint16(1)
	bestError := math.Abs(val)

	var h, k [3]int64
	h[0], k[0] = 0, 1
	h[1], k[1] = 1, 0

	f := val

	for i := 2; i < 100; i++ { // Limit iterations to prevent infinite loops
		a := int64(f)

		// Calculate next convergent
		h[2] = a*h[1] + h[0]
		k[2] = a*k[1] + k[0]

		if h[2] > math.MaxUint16 || k[2] > math.MaxUint16 {
			// This convergent is out of bounds, so the previous one was the best we could do.
			break
		}

		numConv := uint16(h[2])
		denConv := uint16(k[2])

		currentVal := float64(numConv) / float64(denConv)
		currentError := math.Abs(val - currentVal)

		if currentError < bestError {
			bestError = currentError
			bestNum = numConv
			bestDen = denConv
		}

		// Check if we have a perfect approximation
		if f-float64(a) == 0.0 {
			break
		}

		f = 1.0 / (f - float64(a))

		h[0], h[1] = h[1], h[2]
		k[0], k[1] = k[1], k[2]
	}

	return bestNum, bestDen
}

func (self *Image) as_apng() (ans apng.APNG) {
	ans.LoopCount = self.LoopCount
	if self.DefaultImage != nil {
		ans.Frames = append(ans.Frames, apng.Frame{Image: self.DefaultImage, IsDefault: true})
	}
	for i, f := range self.Frames {
		d := apng.Frame{
			DisposeOp: apng.DISPOSE_OP_BACKGROUND, BlendOp: apng.BLEND_OP_OVER, XOffset: f.TopLeft.X, YOffset: f.TopLeft.Y, Image: f.Image,
		}
		if !f.Replace {
			d.BlendOp = apng.BLEND_OP_SOURCE
		}
		d.DelayNumerator, d.DelayDenominator = as_fraction(f.Delay)
		if i+1 < len(self.Frames) {
			nf := self.Frames[i+1]
			switch nf.ComposeOnto {
			case f.Number:
				d.DisposeOp = apng.DISPOSE_OP_NONE
			case 0:
				d.DisposeOp = apng.DISPOSE_OP_BACKGROUND
			case f.ComposeOnto:
				d.DisposeOp = apng.DISPOSE_OP_PREVIOUS
			}
		}
		ans.Frames = append(ans.Frames, d)
	}
	return
}

// Encode this image into a PNG
func (self *Image) EncodeAsPNG(w io.Writer) error {
	if len(self.Frames) < 2 {
		img := self.DefaultImage
		if img == nil {
			img = self.Frames[0].Image
		}
		return png.Encode(w, img)
	}
	// Unfortunately apng.Encode() is buggy or I am getting my dispose op
	// mapping wrong, so coalesce first
	img := self.Clone()
	img.Coalesce()
	return apng.Encode(w, img.as_apng())
}

// Save this image as PNG
func (self *Image) SaveAsPNG(path string, mode fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return self.EncodeAsPNG(f)
}

// Flip all frames horizontally
func (self *Image) FlipH() {
	for _, f := range self.Frames {
		f.Image = FlipH(f.Image)
	}
}

// Flip all frames vertically
func (self *Image) FlipV() {
	for _, f := range self.Frames {
		f.Image = FlipV(f.Image)
	}
}

type rotation struct {
	angle_rads, cos, sin, center_x, center_y float64
}

func new_rotation(angle_deg float64, canvas_rect image.Rectangle) *rotation {
	a := angle_deg * (math.Pi / 180.)
	return &rotation{a, math.Cos(a), math.Sin(a), float64(canvas_rect.Dx()) / 2, float64(canvas_rect.Dy()) / 2}
}

func (r *rotation) apply(p image.Point) image.Point {
	if (p == image.Point{}) {
		return p
	}
	x := float64(p.X) - r.center_x
	y := float64(p.Y) - r.center_y
	rx := x*r.cos - y*r.sin
	ry := x*r.sin + y*r.cos
	return image.Pt(int(rx+r.center_x), int(ry+r.center_y))
}

func (self *Image) Bounds() image.Rectangle {
	if self.DefaultImage != nil {
		return self.DefaultImage.Bounds()
	}
	if len(self.Frames) > 0 {
		return self.Frames[0].Bounds()
	}
	return image.Rect(0, 0, int(self.Metadata.PixelWidth), int(self.Metadata.PixelHeight))
}

// Transpose all frames (flip and rotate 90)
func (self *Image) Transpose() {
	r := new_rotation(90, self.Bounds())
	for _, f := range self.Frames {
		f.Image = Transpose(f.Image)
		f.TopLeft = r.apply(f.TopLeft)
	}
	if self.DefaultImage != nil {
		self.DefaultImage = Transpose(self.DefaultImage)
	}
	self.Metadata.PixelWidth, self.Metadata.PixelHeight = self.Metadata.PixelHeight, self.Metadata.PixelWidth
}

// Transverse all frames (flip and rotate 90)
func (self *Image) Transverse() {
	r := new_rotation(90, self.Bounds())
	for _, f := range self.Frames {
		f.Image = Transverse(f.Image)
		f.TopLeft = r.apply(f.TopLeft)
	}
	if self.DefaultImage != nil {
		self.DefaultImage = Transverse(self.DefaultImage)
	}
	self.Metadata.PixelWidth, self.Metadata.PixelHeight = self.Metadata.PixelHeight, self.Metadata.PixelWidth
}

// Rotate all frames by 90 counter clockwise
func (self *Image) Rotate90() {
	r := new_rotation(90, self.Bounds())
	for _, f := range self.Frames {
		f.Image = Rotate90(f.Image)
		f.TopLeft = r.apply(f.TopLeft)
	}
	if self.DefaultImage != nil {
		self.DefaultImage = Rotate90(self.DefaultImage)
	}
	self.Metadata.PixelWidth, self.Metadata.PixelHeight = self.Metadata.PixelHeight, self.Metadata.PixelWidth
}

// Rotate all frames by 180 counter clockwise
func (self *Image) Rotate180() {
	r := new_rotation(180, self.Bounds())
	for _, f := range self.Frames {
		f.Image = Rotate180(f.Image)
		f.TopLeft = r.apply(f.TopLeft)
	}
	if self.DefaultImage != nil {
		self.DefaultImage = Rotate180(self.DefaultImage)
	}
}

// Rotate all frames by 270 counter clockwise
func (self *Image) Rotate270() {
	r := new_rotation(270, self.Bounds())
	for _, f := range self.Frames {
		f.Image = Rotate270(f.Image)
		f.TopLeft = r.apply(f.TopLeft)
	}
	self.Metadata.PixelWidth, self.Metadata.PixelHeight = self.Metadata.PixelHeight, self.Metadata.PixelWidth
	if self.DefaultImage != nil {
		self.DefaultImage = Rotate270(self.DefaultImage)
	}
}

// Resize all frames to the specified size
func (self *Image) Resize(width, height int, filter ResampleFilter) {
	old_width, old_height := self.Bounds().Dx(), self.Bounds().Dy()
	sx := float64(width) / float64(old_width)
	sy := float64(height) / float64(old_height)
	scaledx := func(x int) int { return int(float64(x) * sx) }
	scaledy := func(y int) int { return int(float64(y) * sy) }
	for i, f := range self.Frames {
		if i == 0 {
			f.Image = ResizeWithOpacity(f.Image, width, height, filter, IsOpaque(f.Image))
		} else {
			f.Image = ResizeWithOpacity(f.Image, scaledx(f.Image.Bounds().Dx()), scaledy(f.Image.Bounds().Dy()), filter, IsOpaque(f.Image))
			f.TopLeft = image.Pt(scaledx(f.TopLeft.X), scaledy(f.TopLeft.Y))
		}
	}
	self.Metadata.PixelWidth, self.Metadata.PixelHeight = uint32(width), uint32(height)
}

// Paste all frames onto the specified background color (OVER alpha blend)
func (img *Image) PasteOntoBackground(bg color.Color) {
	if img.DefaultImage != nil {
		img.DefaultImage = PasteOntoBackground(img.DefaultImage, bg)
	}
	for _, f := range img.Frames {
		f.Image = PasteOntoBackground(f.Image, bg)
	}
}
