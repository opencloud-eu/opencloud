package magick

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/prism/meta"
	"github.com/kovidgoyal/imaging/prism/meta/gifmeta"
	"github.com/kovidgoyal/imaging/prism/meta/icc"
	"github.com/kovidgoyal/imaging/types"
)

var _ = fmt.Print

var MagickExe = sync.OnceValue(func() string {
	ans, err := exec.LookPath("magick")
	if err != nil || ans == "" {
		ans = "magick"
	}
	return ans
})
var HasMagick = sync.OnceValue(func() bool { return MagickExe() != "magick" })

var TempDirInRAMIfPossible = sync.OnceValue(func() string { return get_temp_dir() })

type ImageFrame struct {
	Width, Height, Left, Top int
	Number                   int   // 1-based number
	Compose_onto             int   // number of frame to compose onto
	Delay_ms                 int32 // negative for gapless frame, zero ignored, positive is number of ms
	Replace                  bool  // do a replace rather than an alpha blend
	Is_opaque                bool
	Img                      image.Image
}

func check_resize(frame *ImageFrame, filename string) error {
	// ImageMagick sometimes generates RGBA images smaller than the specified
	// size. See https://github.com/kovidgoyal/kitty/issues/276 for examples
	s, err := os.Stat(filename)
	if err != nil {
		return err
	}
	sz := int(s.Size())
	bytes_per_pixel := 4
	if frame.Is_opaque {
		bytes_per_pixel = 3
	}
	expected_size := bytes_per_pixel * frame.Width * frame.Height
	if sz < expected_size {
		if bytes_per_pixel == 4 && sz == 3*frame.Width*frame.Height {
			frame.Is_opaque = true
			return nil
		}
		missing := expected_size - sz
		if missing%(bytes_per_pixel*frame.Width) != 0 {
			return fmt.Errorf("ImageMagick failed to resize correctly. It generated %d < %d of data (w=%d h=%d bpp=%d frame-number: %d)", sz, expected_size, frame.Width, frame.Height, bytes_per_pixel, frame.Number)
		}
		frame.Height -= missing / (bytes_per_pixel * frame.Width)
	}
	return nil
}

func RunMagick(i *input, cmd []string) ([]byte, error) {
	cmd = append([]string{MagickExe()}, cmd...)
	c := exec.Command(cmd[0], cmd[1:]...)
	if i.os_file != nil {
		c.ExtraFiles = append(c.ExtraFiles, i.os_file)
	}
	output, err := c.Output()
	if err != nil {
		var exit_err *exec.ExitError
		if errors.As(err, &exit_err) {
			return nil, fmt.Errorf("Running the command: %s\nFailed with error:\n%s", strings.Join(cmd, " "), string(exit_err.Stderr))
		}
		return nil, fmt.Errorf("Could not find the program: %#v. Is ImageMagick installed and in your PATH?", cmd[0])
	}
	return output, nil
}

type IdentifyOutput struct {
	Fmt, Canvas, Transparency, Gap, Index, Size, Dpi, Dispose, Orientation, Colorspace string
}

type DisposeOp int

const (
	DisposeNone DisposeOp = iota
	DisposeBackground
	DisposePrevious
)

type IdentifyRecord struct {
	Fmt_uppercase      string
	Gap                int
	Canvas             struct{ Width, Height, Left, Top int }
	Width, Height      int
	Dpi                struct{ X, Y float64 }
	Index              int
	Is_opaque          bool
	Needs_blend        bool
	Disposal           DisposeOp
	Dimensions_swapped bool
	ColorSpace         string
}

func parse_identify_record(ans *IdentifyRecord, raw *IdentifyOutput) (err error) {
	ans.Fmt_uppercase = strings.ToUpper(raw.Fmt)
	if raw.Gap != "" {
		ans.Gap, err = strconv.Atoi(raw.Gap)
		if err != nil {
			return fmt.Errorf("Invalid gap value in identify output: %s", raw.Gap)
		}
		ans.Gap = max(0, ans.Gap)
	}
	area, pos, found := strings.Cut(raw.Canvas, "+")
	ok := false
	if found {
		w, h, found := strings.Cut(area, "x")
		if found {
			ans.Canvas.Width, err = strconv.Atoi(w)
			if err == nil {
				ans.Canvas.Height, err = strconv.Atoi(h)
				if err == nil {
					x, y, found := strings.Cut(pos, "+")
					if found {
						ans.Canvas.Left, err = strconv.Atoi(x)
						if err == nil {
							if ans.Canvas.Top, err = strconv.Atoi(y); err == nil {
								ok = true
							}
						}
					}
				}
			}
		}
	}
	if !ok {
		return fmt.Errorf("Invalid canvas value in identify output: %s", raw.Canvas)
	}
	w, h, found := strings.Cut(raw.Size, "x")
	ok = false
	if found {
		ans.Width, err = strconv.Atoi(w)
		if err == nil {
			if ans.Height, err = strconv.Atoi(h); err == nil {
				ok = true
			}
		}
	}
	if !ok {
		return fmt.Errorf("Invalid size value in identify output: %s", raw.Size)
	}
	x, y, found := strings.Cut(raw.Dpi, "x")
	ok = false
	if found {
		ans.Dpi.X, err = strconv.ParseFloat(x, 64)
		if err == nil {
			if ans.Dpi.Y, err = strconv.ParseFloat(y, 64); err == nil {
				ok = true
			}
		}
	}
	if !ok {
		return fmt.Errorf("Invalid dpi value in identify output: %s", raw.Dpi)
	}
	ans.Index, err = strconv.Atoi(raw.Index)
	if err != nil {
		return fmt.Errorf("Invalid index value in identify output: %s", raw.Index)
	}
	q := strings.ToLower(raw.Transparency)
	if q == "blend" || q == "true" {
		ans.Is_opaque = false
	} else {
		ans.Is_opaque = true
	}
	ans.Needs_blend = q == "blend"
	switch strings.ToLower(raw.Dispose) {
	case "none", "undefined":
		ans.Disposal = DisposeNone
	case "background":
		ans.Disposal = DisposeBackground
	case "previous":
		ans.Disposal = DisposePrevious
	default:
		return fmt.Errorf("Invalid value for dispose: %s", raw.Dispose)
	}
	ans.ColorSpace = raw.Colorspace
	switch raw.Orientation {
	case "5", "6", "7", "8":
		ans.Dimensions_swapped = true
	}
	if ans.Dimensions_swapped {
		ans.Canvas.Width, ans.Canvas.Height = ans.Canvas.Height, ans.Canvas.Width
		ans.Width, ans.Height = ans.Height, ans.Width
	}

	return
}

func identify(path *input) (ans []IdentifyRecord, err error) {
	cmd := []string{"identify"}
	q := `{"fmt":"%m","canvas":"%g","transparency":"%A","gap":"%T","index":"%p","size":"%wx%h",` +
		`"dpi":"%xx%y","dispose":"%D","orientation":"%[EXIF:Orientation]","colorspace":"%[colorspace]"},`
	cmd = append(cmd, "-format", q, "--", path.arg)
	output, err := RunMagick(path, cmd)
	if err != nil {
		return nil, fmt.Errorf("Failed to identify image at path: %s with error: %w", path, err)
	}
	output = bytes.TrimRight(bytes.TrimSpace(output), ",")
	raw_json := make([]byte, 0, len(output)+2)
	raw_json = append(raw_json, '[')
	raw_json = append(raw_json, output...)
	raw_json = append(raw_json, ']')
	var records []IdentifyOutput
	err = json.Unmarshal(raw_json, &records)
	if err != nil {
		return nil, fmt.Errorf("The ImageMagick identify program returned malformed output for the image at path: %s, with error: %w", path, err)
	}
	ans = make([]IdentifyRecord, len(records))
	for i, rec := range records {
		err = parse_identify_record(&ans[i], &rec)
		if err != nil {
			return nil, err
		}
	}
	return ans, nil
}

type RenderOptions struct {
	Background             *color.RGBA64
	ResizeTo               image.Point
	OnlyFirstFrame         bool
	AutoOrient             bool
	ToSRGB                 bool
	Transform              types.TransformType
	RenderingIntent        icc.RenderingIntent
	BlackpointCompensation bool
}

func rgba64ToImageMagick(c_ color.RGBA64) string {
	c := color.NRGBA64Model.Convert(c_).(color.NRGBA64)
	rPercent := float64(c.R) / 65535.0 * 100.0
	gPercent := float64(c.G) / 65535.0 * 100.0
	bPercent := float64(c.B) / 65535.0 * 100.0
	alpha := float64(c.A) / 65535.0
	return fmt.Sprintf("rgba(%.3f%%,%.3f%%,%.3f%%,%.4f)", rPercent, gPercent, bPercent, alpha)
}

func is_not_srgb(name string) bool {
	return name != "" && strings.ToUpper(name) != "SRGB"
}

func render(path *input, ro *RenderOptions, is_srgb bool, frames []IdentifyRecord) (ans []*ImageFrame, err error) {
	cmd := []string{}
	add_alpha_remove := false
	if ro.Background == nil {
		cmd = append(cmd, "-background", "none")
	} else {
		if ro.Background.A == 0xffff {
			n := nrgb.Model.Convert(*ro.Background).(nrgb.Color)
			add_alpha_remove = true
			cmd = append(cmd, "-background", n.AsSharp())
		} else {
			cmd = append(cmd, "-background", rgba64ToImageMagick(*ro.Background))
		}
	}
	cpath := path.arg
	if ro.OnlyFirstFrame {
		cpath += "[0]"
	}
	has_multiple_frames := len(frames) > 1
	get_multiple_frames := has_multiple_frames && !ro.OnlyFirstFrame
	cmd = append(cmd, "--", cpath)
	if ro.AutoOrient {
		cmd = append(cmd, "-auto-orient")
	}
	if add_alpha_remove {
		cmd = append(cmd, "-alpha", "remove")
	} else if ro.Background != nil {
		cmd = append(cmd, "-flatten")
	}
	switch ro.Transform {
	case types.FlipHTransform:
		cmd = append(cmd, "-flop")
	case types.FlipVTransform:
		cmd = append(cmd, "-flip")
	case types.TransposeTransform:
		cmd = append(cmd, "-transpose")
	case types.TransverseTransform:
		cmd = append(cmd, "-transverse")
	case types.Rotate90Transform:
		cmd = append(cmd, "-rotate", "-90")
	case types.Rotate180Transform:
		cmd = append(cmd, "-rotate", "180")
	case types.Rotate270Transform:
		cmd = append(cmd, "-rotate", "-270")
	}
	tdir, err := os.MkdirTemp(TempDirInRAMIfPossible(), "")
	if err != nil {
		err = fmt.Errorf("failed to create temporary directory to hold ImageMagick output with error: %w", err)
		return
	}
	defer os.RemoveAll(tdir)
	if ro.ToSRGB && !is_srgb {
		profile_path := filepath.Join(tdir, "sRGB.icc")
		if err = os.WriteFile(profile_path, icc.Srgb_xyz_profile_data, 0o666); err != nil {
			return nil, fmt.Errorf("failed to create temporary file with profile for ImageMagick with error: %w", err)
		}

		cmd = append(cmd, icc.IfElse(ro.BlackpointCompensation, "-", "+")+"black-point-compensation")
		cmd = append(cmd, "-intent", ro.RenderingIntent.String())
		cmd = append(cmd, "-profile", profile_path)
	}
	if ro.ResizeTo.X > 0 {
		rcmd := []string{"-resize", fmt.Sprintf("%dx%d!", ro.ResizeTo.X, ro.ResizeTo.Y)}
		if get_multiple_frames {
			cmd = append(cmd, "-coalesce")
			cmd = append(cmd, rcmd...)
			cmd = append(cmd, "-deconstruct")
		} else {
			cmd = append(cmd, rcmd...)
		}
	}
	cmd = append(cmd, "-depth", "8", "-set", "filename:f", "%w-%h-%g-%p")
	if get_multiple_frames {
		cmd = append(cmd, "+adjoin")
	}
	mode := "rgba"
	if frames[0].Is_opaque {
		mode = "rgb"
	}
	cmd = append(cmd, filepath.Join(tdir, "im-%[filename:f]."+mode))
	_, err = RunMagick(path, cmd)
	if err != nil {
		return
	}
	entries, err := os.ReadDir(tdir)
	if err != nil {
		err = fmt.Errorf("Failed to read temp dir used to store ImageMagick output with error: %w", err)
		return
	}
	gaps := make([]int, len(frames))
	for i, frame := range frames {
		gaps[i] = frame.Gap
	}
	// although ImageMagick *might* be already taking care of this adjustment,
	// I dont know for sure, so do it anyway.
	min_gap := gifmeta.CalcMinimumGap(gaps)
	for _, entry := range entries {
		fname := entry.Name()
		p, _, _ := strings.Cut(fname, ".")
		parts := strings.Split(p, "-")
		if len(parts) < 5 {
			continue
		}
		index, cerr := strconv.Atoi(parts[len(parts)-1])
		if cerr != nil || index < 0 || index >= len(frames) {
			continue
		}
		width, cerr := strconv.Atoi(parts[1])
		if cerr != nil {
			continue
		}
		height, cerr := strconv.Atoi(parts[2])
		if cerr != nil {
			continue
		}
		_, pos, found := strings.Cut(parts[3], "+")
		if !found {
			continue
		}
		px, py, found := strings.Cut(pos, "+")
		if !found {
			// ImageMagick is a buggy POS
			px, py = "0", "0"
		}
		x, cerr := strconv.Atoi(px)
		if cerr != nil {
			continue
		}
		y, cerr := strconv.Atoi(py)
		if cerr != nil {
			continue
		}
		identify_data := frames[index]
		path := filepath.Join(tdir, fname)
		frame := ImageFrame{
			Number: index + 1, Width: width, Height: height, Left: x, Top: y, Is_opaque: identify_data.Is_opaque,
		}
		frame.Delay_ms = int32(max(min_gap, identify_data.Gap) * 10)
		err = check_resize(&frame, path)
		if err != nil {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("Failed to read temp file for image %#v with error: %w", path, err)
		}
		dest_rect := image.Rect(0, 0, frame.Width, frame.Height)
		if frame.Is_opaque {
			frame.Img = &nrgb.Image{Pix: data, Stride: frame.Width * 3, Rect: dest_rect}
		} else {
			frame.Img = &image.NRGBA{Pix: data, Stride: frame.Width * 4, Rect: dest_rect}
		}
		ans = append(ans, &frame)
	}
	if len(ans) < len(frames) {
		err = fmt.Errorf("Failed to render %d out of %d frames", len(frames)-len(ans), len(frames))
		return
	}
	slices.SortFunc(ans, func(a, b *ImageFrame) int { return a.Number - b.Number })
	prev_disposal := DisposeBackground
	prev_compose_onto := 0
	for i, frame := range ans {
		switch prev_disposal {
		case DisposeNone:
			frame.Compose_onto = frame.Number - 1
		case DisposePrevious:
			frame.Compose_onto = prev_compose_onto
		}
		prev_disposal, prev_compose_onto = frames[i].Disposal, frame.Compose_onto
	}
	return
}

type Image struct {
	Frames           []*ImageFrame
	Format_uppercase string
}

type input struct {
	arg          string
	os_file      *os.File
	needs_close  bool
	needs_remove string
}

func (i input) String() string {
	return i.arg
}

func magick_input_path(i *types.Input) (inp *input, err error) {
	if i.Path != "" {
		return &input{arg: i.Path}, nil
	}
	r := i.Reader
	if s, ok := r.(*os.File); ok {
		if _, serr := s.Seek(0, io.SeekCurrent); serr == nil {
			if s.Name() != "" {
				if _, serr := os.Stat(s.Name()); serr == nil {
					return &input{arg: s.Name()}, nil
				}
			}
			if runtime.GOOS != "windows" {
				return &input{arg: "fd:3", os_file: s}, nil
			}
		}
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if f, merr := memfd(data); merr == nil {
		return &input{arg: "fd:3", os_file: f, needs_close: true}, nil
	}
	f, err := os.CreateTemp(TempDirInRAMIfPossible(), "")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return nil, err
	}
	return &input{arg: f.Name(), needs_remove: f.Name()}, nil
}

func OpenAll(input *types.Input, md *meta.Data, callback func(w, h int) RenderOptions) (ans *Image, err error) {
	if !HasMagick() {
		return nil, fmt.Errorf("the magick command as not found in PATH")
	}
	// ImageMagick needs to be told explicitly to use APNG otherwise it only returns the first frame
	i, err := magick_input_path(input)
	if i.os_file != nil && i.needs_close {
		defer i.os_file.Close()
	}
	if i.needs_remove != "" {
		defer os.Remove(i.needs_remove)
	}
	if err != nil {
		return nil, err
	}
	is_apng := md != nil && md.Format == types.PNG && md.HasFrames
	if is_apng {
		i.arg = "APNG:" + i.arg
	}
	identify_records, err := identify(i)
	if err != nil {
		return nil, err
	}
	is_srgb := !is_not_srgb(identify_records[0].ColorSpace)
	if is_srgb && md != nil {
		// ImageMagick is a PoS that cant identify profiles
		is_srgb = md.IsSRGB()
	}
	ro := callback(identify_records[0].Canvas.Width, identify_records[0].Canvas.Height)
	frames, err := render(i, &ro, is_srgb, identify_records)
	if err != nil {
		return nil, err
	}
	ans = &Image{
		Format_uppercase: identify_records[0].Fmt_uppercase, Frames: frames,
	}
	return ans, nil
}
