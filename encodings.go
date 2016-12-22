// Implementation of RFC 6143 §7.7 Encodings.

package vnc

import (
	"bytes"
	"fmt"
)

const (
	Raw               = int32(0)
	CopyRect          = int32(1)
	RRE               = int32(2)
	Hextile           = int32(5)
	TRLE              = int32(15)
	ZRLE              = int32(16)
	ColorPseudo       = int32(-239)
	DesktopSizePseudo = int32(-223)
)

// An Encoding implements a method for encoding pixel data that is
// sent by the server to the client.
type Encoding interface {
	// The number that uniquely identifies this encoding type.
	Type() int32

	// Read the contents of the encoded pixel data from the reader.
	// This should return a new Encoding implementation that contains
	// the proper data.
	Read(*ClientConn, *Rectangle) (Encoding, []byte, error)

	// Marshal implements the Marshaler interface.
	Marshal() ([]byte, error)
}

type Encodings []Encoding

func (e Encodings) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)

	for _, enc := range e {
		if err := buf.Write(enc.Type()); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// RawEncoding is raw pixel data sent by the server.
// See RFC 6143 §7.7.1.
type RawEncoding struct {
	Colors []Color
}

func (*RawEncoding) Type() int32 {
	return Raw
}

func (e *RawEncoding) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)

	for _, c := range e.Colors {
		bytes, err := c.Marshal()
		if err != nil {
			return nil, err
		}
		buf.WriteBytes(bytes)
	}
	return buf.Bytes(), nil
}

func convert(data []byte, pf *PixelFormat) (r,g,b,a uint8) {
    a=255
    order := pf.order()

    var pixel uint32
    pixel = 0
    switch pf.BPP {
    case 8:
        pixel = uint32(data[0])
    case 16:
        pixel = uint32(order.Uint16(data))
    case 32:
        pixel = order.Uint32(data)
    }

    if pf.TrueColor == RFBTrue {
        r = uint8((pixel >> pf.RedShift) & uint32(pf.RedMax))
        g = uint8((pixel >> pf.GreenShift) & uint32(pf.GreenMax))
        b = uint8((pixel >> pf.BlueShift) & uint32(pf.BlueMax))
    } else {
        r = 255
        b = 0
        g = 0
        //*c = c.cm[pixel]
        //c.cmIndex = pixel
    }

    return
}

func (*RawEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, []byte, error) {
	// if c.debug {
	// 	log.Println("RawEncoding.Read()")
	// 	rect.DebugPrint()
	// }


	bytesPerPixel := int(c.pixelFormat.BPP / 8)
	n := rect.Area() * bytesPerPixel
    b := make([]byte, n)

	if err := c.readByteMsg(&b); err != nil {
		return nil, nil, fmt.Errorf("unable to read rectangle with raw encoding: %v", err)
	}

    buf := bytes.NewBuffer(b)

	//colors := make([]Color, rect.Area())
    Pix := make([]byte, rect.Area()*4)
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
            data := buf.Next(bytesPerPixel)
            r,g,b,a := convert(data, &c.pixelFormat)
            offset := 4*(int(y)*int(rect.Width)+int(x))
			//color := NewColor(&c.pixelFormat, &c.colorMap)
			//if err := color.Unmarshal(data); err != nil {
				//return nil, nil, err
			//}
            
            Pix[offset] = r //uint8(color.R)
            Pix[offset+1] = g //uint8(color.G)
            Pix[offset+2] = b //uint8(color.B)
            Pix[offset+3] = a //255
			//colors[int(y)*int(rect.Width)+int(x)] = *color
		}
	}

	//return &RawEncoding{colors}, Pix,  nil
	return &RawEncoding{}, Pix,  nil
}

// DesktopSizePseudoEncoding enables desktop resize support.
// See RFC 6143 §7.8.2.
type DesktopSizePseudoEncoding struct{}

func (*DesktopSizePseudoEncoding) Type() int32 {
	return DesktopSizePseudo
}

func (*DesktopSizePseudoEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, []byte, error) {
	c.fbWidth = rect.Width
	c.fbHeight = rect.Height

	return &DesktopSizePseudoEncoding{}, nil, nil
}

func (e *DesktopSizePseudoEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}
