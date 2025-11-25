package netpbm

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"

	"github.com/kovidgoyal/go-parallel"
	"github.com/kovidgoyal/imaging/nrgb"
	"github.com/kovidgoyal/imaging/types"
)

var _ = fmt.Print

// skip_comments reads ahead past any comment lines (starting with #) and returns the first non-comment, non-empty line.
func skip_comments(br *bufio.Reader) (string, error) {
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line, nil
	}
}

type data_type int

const (
	rgb data_type = iota
	blackwhite
	grayscale
)

type header struct {
	format                      string
	width, height, num_channels uint
	maxval                      uint32
	has_alpha                   bool
	data_type                   data_type
}

func (h header) bytes_per_channel() uint {
	if h.maxval > 255 {
		return 2
	}
	return 1
}

func (h header) num_bytes_per_pixel() uint {
	return h.num_channels * h.bytes_per_channel()
}

func read_ppm_header(br *bufio.Reader, magic string) (ans header, err error) {
	ans.format = magic
	required_num_fields := 3
	switch magic {
	case "P1", "P4":
		ans.data_type = blackwhite
		ans.num_channels = 1
		ans.maxval = 1
		required_num_fields = 2
	case "P2", "P5":
		ans.data_type = grayscale
		ans.num_channels = 1
	default:
		ans.data_type = rgb
		ans.num_channels = 3
	}
	var fields []uint
	for len(fields) < required_num_fields {
		var line string
		if line, err = skip_comments(br); err != nil {
			return
		}
		for x := range strings.FieldsSeq(line) {
			var val uint64
			if val, err = strconv.ParseUint(x, 10, 0); err != nil {
				return
			}
			fields = append(fields, uint(val))
		}
	}
	ans.width = fields[0]
	ans.height = fields[1]
	if required_num_fields > 2 {
		if fields[2] > 65535 {
			return ans, fmt.Errorf("header specifies a maximum value %d larger than 65535", ans.maxval)
		}
		ans.maxval = uint32(fields[2])
	}
	if ans.maxval > 65535 {
		return ans, fmt.Errorf("header specifies a maximum value %d larger than 65535", ans.maxval)
	}
	return
}

func read_pam_header(br *bufio.Reader) (ans header, err error) {
	ans.format = "P7"
	ans.data_type = rgb
	ans.num_channels = 3
	for {
		line, err := skip_comments(br)
		if err != nil {
			return ans, err
		}
		if line == "ENDHDR" {
			break
		}
		prefix, payload, found := strings.Cut(line, " ")
		if !found {
			return ans, fmt.Errorf("invalid line in header: %#v", line)
		}
		switch prefix {
		case "WIDTH":
			w, err := strconv.ParseUint(payload, 10, 0)
			if err != nil {
				return ans, fmt.Errorf("invalid width %#v in header: %w", payload, err)
			}
			ans.width = uint(w)
		case "HEIGHT":
			w, err := strconv.ParseUint(payload, 10, 0)
			if err != nil {
				return ans, fmt.Errorf("invalid height %#v in header: %w", payload, err)
			}
			ans.height = uint(w)
		case "MAXVAL":
			w, err := strconv.ParseUint(payload, 10, 0)
			if err != nil {
				return ans, fmt.Errorf("invalid maxval %#v in header: %w", payload, err)
			}
			ans.maxval = uint32(w)
		case "DEPTH":
			w, err := strconv.ParseUint(payload, 10, 0)
			if err != nil {
				return ans, fmt.Errorf("invalid depth %#v in header: %w", payload, err)
			}
			if w == 0 || w > 4 {
				return ans, fmt.Errorf("invalid depth %d in header", w)
			}
			ans.num_channels = uint(w)
		case "TUPLTYPE":
			switch payload {
			case "BLACKANDWHITE":
				ans.data_type = blackwhite
			case "BLACKANDWHITE_ALPHA":
				ans.has_alpha = true
				ans.data_type = blackwhite
			case "GRAYSCALE":
				ans.data_type = grayscale
			case "GRAYSCALE_ALPHA":
				ans.has_alpha = true
				ans.data_type = grayscale
			case "RGB":
			case "RGB_ALPHA":
				ans.has_alpha = true
			default:
				return ans, fmt.Errorf("invalid TUPLTYPE in header: %#v", payload)
			}
		}
	}
	if ans.width == 0 || ans.height == 0 || ans.maxval == 0 {
		return ans, fmt.Errorf("header does not specify width, height and maximum value")
	}
	ok := true
	switch ans.data_type {
	case rgb:
		ok = (!ans.has_alpha && ans.num_channels == 3) || (ans.has_alpha && ans.num_channels == 4)
	case blackwhite, grayscale:
		ok = (!ans.has_alpha && ans.num_channels == 1) || (ans.has_alpha && ans.num_channels == 2)
	}
	if !ok {
		return ans, fmt.Errorf("header specified depth: %d does not match TUPLTYPE", ans.num_channels)
	}
	return
}

func read_header(br *bufio.Reader) (ans header, err error) {
	b := []byte{0, 0}
	if _, err = io.ReadFull(br, b); err != nil {
		return ans, err
	}
	magic := string(b)
	switch magic {
	case "P1", "P2", "P3", "P4", "P5", "P6":
		return read_ppm_header(br, magic)
	case "P7":
		return read_pam_header(br)
	default:
		err = fmt.Errorf("unsupported netPBM format: %#v", magic)
		return
	}
}

func ascii_range_over_values(br *bufio.Reader, h header, callback func(uint32, []uint8) []uint8) (ans []uint8, err error) {
	anssz := h.width * h.height * h.num_bytes_per_pixel()
	ans = make([]uint8, 0, anssz)
	for uint(len(ans)) < anssz {
		token, err := br.ReadString(' ')
		if err != nil && err != io.EOF {
			return nil, err
		}
		for field := range strings.FieldsSeq(token) {
			if val, perr := strconv.ParseUint(field, 10, 16); perr == nil {
				ans = callback(uint32(val), ans)
			}
		}
		if err == io.EOF {
			break
		}
	}
	return
}

func decode_rgb_ascii(br *bufio.Reader, h header) (ans []byte, err error) {
	mult := uint32(255)
	if h.maxval > 255 {
		mult = 65535
	}
	anssz := h.width * h.height * h.num_bytes_per_pixel()
	if mult == 255 {
		ans, err = ascii_range_over_values(br, h, func(val uint32, ans []uint8) []uint8 {
			ch := (uint32(val) * mult) / h.maxval
			return append(ans, uint8(ch))
		})
	} else {
		ans, err = ascii_range_over_values(br, h, func(val uint32, ans []uint8) []uint8 {
			ch := (uint32(val) * mult) / h.maxval
			ans = append(ans, uint8(ch))
			if len(ans)%6 == 0 { // alpha is always 255
				ans = append(ans, 255, 255)
			}
			return ans
		})
	}
	if err != nil {
		return nil, err
	}
	if uint(len(ans)) < anssz {
		return nil, errors.New("insufficient color data present in PPM file")
	}
	return
}

func DecodeConfigAndFormat(r io.Reader) (cfg image.Config, fmt types.Format, err error) {
	br := bufio.NewReader(r)
	h, err := read_header(br)
	if err != nil {
		return cfg, types.UNKNOWN, err
	}
	cfg.Width = int(h.width)
	cfg.Height = int(h.height)
	cfg.ColorModel = nrgb.Model
	switch h.data_type {
	case blackwhite, grayscale:
		if h.has_alpha {
			if h.maxval > 255 {
				cfg.ColorModel = color.NRGBA64Model
			} else {
				cfg.ColorModel = color.NRGBAModel
			}
		} else {
			if h.maxval > 255 {
				cfg.ColorModel = color.Gray16Model
			} else {
				cfg.ColorModel = color.GrayModel
			}
		}
	default:
		if h.has_alpha {
			if h.maxval > 255 {
				cfg.ColorModel = color.NRGBA64Model
			} else {
				cfg.ColorModel = color.NRGBAModel
			}
		} else {
			if h.maxval > 255 {
				cfg.ColorModel = color.NRGBA64Model
			} else {
				cfg.ColorModel = nrgb.Model
			}
		}
	}
	switch h.format {
	case "P7":
		fmt = types.PAM
	case "P1", "P4":
		fmt = types.PBM
	case "P2", "P5":
		fmt = types.PGM
	case "P3", "P6":
		fmt = types.PPM
	}
	return
}

func DecodeConfig(r io.Reader) (cfg image.Config, err error) {
	cfg, _, err = DecodeConfigAndFormat(r)
	return
}

func decode_black_white_ascii(br *bufio.Reader, h header) (img image.Image, err error) {
	r := image.Rect(0, 0, int(h.width), int(h.height))
	g := &image.Gray{Stride: r.Dx(), Rect: r}
	g.Pix, err = ascii_range_over_values(br, h, func(val uint32, ans []uint8) []uint8 {
		var c uint8 = 255 * uint8(1-(val&1))
		return append(ans, c)
	})
	return g, err
}

func decode_grayscale_ascii(br *bufio.Reader, h header) (img image.Image, err error) {
	r := image.Rect(0, 0, int(h.width), int(h.height))
	if h.maxval > 255 {
		g := &image.Gray16{Stride: 2 * r.Dx(), Rect: r}
		g.Pix, err = ascii_range_over_values(br, h, func(val uint32, ans []uint8) []uint8 {
			c := uint16(val * 65535 / h.maxval)
			return append(ans, uint8(c>>8), uint8(c))
		})
		return g, err
	} else {
		g := &image.Gray{Stride: r.Dx(), Rect: r}
		g.Pix, err = ascii_range_over_values(br, h, func(val uint32, ans []uint8) []uint8 {
			c := uint8(val * 255 / h.maxval)
			return append(ans, c)
		})
		return g, err
	}
}

// Consume whitespace after header (per spec, it's a single whitespace, but can be more)
func skip_whitespace_before_pixel_data(br *bufio.Reader, num_of_bytes_needed uint) ([]uint8, error) {
	for {
		b, err := br.Peek(1)
		if err != nil {
			return nil, err
		}
		if b[0] == '\n' || b[0] == '\r' || b[0] == '\t' || b[0] == ' ' {
			br.ReadByte()
		} else {
			break
		}
	}
	ans := make([]byte, num_of_bytes_needed)
	_, err := io.ReadFull(br, ans)
	return ans, err
}

func rescale(v uint32, num, den uint32) uint32 {
	return (v * num) / den
}

func rescale_binary_data(b []uint8, num, den uint32) error {
	return parallel.Run_in_parallel_over_range(0, func(start, end int) {
		for i := start; i < end; i++ {
			b[i] = uint8(rescale(uint32(b[i]), num, den))
		}
	}, 0, len(b))
}

func rescale_binary_data16(b []uint8, num, den uint32) error {
	if len(b)&1 != 0 {
		return fmt.Errorf("pixel data is not a multiple of two but uses 16 bits per channel")
	}
	return parallel.Run_in_parallel_over_range(0, func(start, end int) {
		start *= 2
		end *= 2
		for i := start; i < end; i += 2 {
			v := uint32((uint16(b[i]) << 8) | uint16(b[i+1]))
			v = rescale(v, num, den)
			b[i] = uint8(v >> 8)
			b[i+1] = uint8(v)
		}
	}, 0, len(b)/2)
}

func decode_binary_data(br *bufio.Reader, h header) (ans image.Image, err error) {
	var binary_data []uint8
	if binary_data, err = skip_whitespace_before_pixel_data(br, h.width*h.height*h.num_bytes_per_pixel()); err != nil {
		return
	}
	if n := h.num_bytes_per_pixel() * h.width * h.height; uint(len(binary_data)) < n {
		return nil, fmt.Errorf(
			"insufficient pixel data for image area and num_channels (%d): %f < %d",
			h.num_channels, float64(len(binary_data))/float64(h.width*h.height), n/(h.width*h.height))
	}
	switch {
	case h.maxval < 255:
		if err = rescale_binary_data(binary_data, 255, h.maxval); err != nil {
			return nil, err
		}
	case 255 < h.maxval && h.maxval < 65535:
		if err = rescale_binary_data16(binary_data, 65535, h.maxval); err != nil {
			return nil, err
		}
	}

	r := image.Rect(0, 0, int(h.width), int(h.height))
	switch h.num_channels {
	case 1:
		// bw or gray without alpha
		if h.maxval > 255 {
			return &image.Gray16{Rect: r, Stride: r.Dx() * 2, Pix: binary_data}, nil
		}
		return &image.Gray{Rect: r, Stride: r.Dx(), Pix: binary_data}, nil
	case 2:
		// bw or gray with alpha
		if h.maxval > 255 {
			g := image.NewNRGBA64(r)
			b := g.Pix
			if err = parallel.Run_in_parallel_over_range(0, func(start, end int) {
				for i := start; i < end; i++ {
					src := binary_data[i*4 : i*4+4]
					dest := b[i*8 : i*8+8]
					gray1, gray2 := src[0], src[1]
					dest[0], dest[1], dest[2], dest[3], dest[4], dest[5] = gray1, gray2, gray1, gray2, gray1, gray2
					dest[6], dest[7] = src[2], src[3]
				}
			}, 0, int(h.width*h.height)); err != nil {
				return nil, err
			}
		}
		g := image.NewNRGBA(r)
		b := g.Pix
		if err = parallel.Run_in_parallel_over_range(0, func(start, end int) {
			for i := start; i < end; i++ {
				src := binary_data[i*2 : i*2+2]
				dest := b[i*4 : i*4+4]
				dest[0], dest[1], dest[2], dest[3] = src[0], src[0], src[0], src[1]
			}
		}, 0, int(h.width*h.height)); err != nil {
			return nil, err
		}
		return g, nil
	case 3:
		// RGB without alpha
		if h.maxval > 255 {
			g := image.NewNRGBA64(r)
			b := g.Pix
			if err = parallel.Run_in_parallel_over_range(0, func(start, end int) {
				for i := start; i < end; i++ {
					src := binary_data[i*6 : i*6+6]
					dest := b[i*8 : i*8+8]
					copy(dest[:6], src)
					dest[6], dest[7] = 255, 255
				}
			}, 0, int(h.width*h.height)); err != nil {
				return nil, err
			}
			return g, nil
		}
		return nrgb.NewNRGBWithContiguousRGBPixels(binary_data, 0, 0, r.Dx(), r.Dy())
	case 4:
		// RGB with alpha
		if h.maxval <= 255 {
			return &image.NRGBA{Rect: r, Stride: r.Dx() * int(h.num_bytes_per_pixel()), Pix: binary_data}, nil
		}
		return &image.NRGBA64{Rect: r, Stride: r.Dx() * int(h.num_bytes_per_pixel()), Pix: binary_data}, nil
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", h.num_channels)
	}
}

// Decode decodes a PPM image from r and returns it as an image.Image.
// Supports both P3 (ASCII) and P6 (binary) variants.
func Decode(r io.Reader) (img image.Image, err error) {
	br := bufio.NewReader(r)
	h, err := read_header(br)
	if err != nil {
		return nil, err
	}
	var binary_data []uint8
	switch h.format {
	case "P1":
		return decode_black_white_ascii(br, h)
	case "P2":
		return decode_grayscale_ascii(br, h)
	case "P3":
		vals, err := decode_rgb_ascii(br, h)
		if err != nil {
			return nil, err
		}
		if h.maxval <= 255 {
			return nrgb.NewNRGBWithContiguousRGBPixels(vals, 0, 0, int(h.width), int(h.height))
		}
		return &image.NRGBA64{Pix: vals, Stride: int(h.width) * 8, Rect: image.Rect(0, 0, int(h.width), int(h.height))}, nil
	case "P4":
		bytes_per_row := (h.width + 7) / 8
		if binary_data, err = skip_whitespace_before_pixel_data(br, h.height*bytes_per_row); err != nil {
			return nil, err
		}
		ans := image.NewGray(image.Rect(0, 0, int(h.width), int(h.height)))
		i := 0
		for range h.height {
			for x := range h.width {
				byteIdx := x / 8
				bitIdx := 7 - uint(x%8)
				bit := (binary_data[byteIdx] >> bitIdx) & 1
				ans.Pix[i] = (1 - bit) * 255
				i++
			}
			binary_data = binary_data[bytes_per_row:]
		}
		if len(binary_data) > 0 {
			return nil, fmt.Errorf("insufficient color data in netPBM file, need %d more bytes", len(binary_data))
		}
		return ans, nil
	case "P5", "P6", "P7":
		return decode_binary_data(br, h)
	default:
		return nil, fmt.Errorf("invalid format for PPM: %#v", h.format)
	}
}

// Register this decoder with Go's image package
func init() {
	image.RegisterFormat("pbm", "P1", Decode, DecodeConfig)
	image.RegisterFormat("pgm", "P2", Decode, DecodeConfig)
	image.RegisterFormat("ppm", "P3", Decode, DecodeConfig)
	image.RegisterFormat("pbm", "P4", Decode, DecodeConfig)
	image.RegisterFormat("pgm", "P5", Decode, DecodeConfig)
	image.RegisterFormat("ppm", "P6", Decode, DecodeConfig)
	image.RegisterFormat("pam", "P7", Decode, DecodeConfig)
}
