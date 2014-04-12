package heartbleed_dtls

import (
	"bytes"
    "crypto/rand"
	"encoding/binary"
	"encoding/hex"
    "fmt"
    "time"
)

var ContentTypeChangeCypherSpec = []byte{20}
var ContentTypeAlter = []byte{21}
var ContentTypeHandshake = []byte{22}
var ContentTypeApplicationData = []byte{23}
var HandshakeDTLSVersion = []byte{0xfe, 0xff}

const (
    DTLSv10 = 0xfeff
    DTLSv12 = 0xfefd
)

var ClientHelloHandshakePart1 = []byte{
    0x01,             // Handshake Type: Client Hello (1)
    // Length [3 bytes]
}

var ClientHelloHandshakePart2 = []byte{
    // Message Sequence [2 bytes]
    0x00, 0x00, 0x00, // Fragment Offset: 0
    // Fragment Length [3 bytes]
}

var ClientHelloHandshakePart3 = []byte{
    // version [2 bytes]
    // epoch [4 bytes]
    // random [28 bytes]
    0x00,                         // Session ID Length: 0
    0x00,                         // Cookie Length: 0
    0x00, 0x40,                   // Cipher Suites Length: 64
    0xc0, 0x14, 0xc0, 0x0a, 0x00, 0x39, 0x00, 0x38 , 0x00, 0x88, 0x00, 0x87,
    0xc0, 0x0f, 0xc0, 0x05, 0x00, 0x35, 0x00, 0x84, 0xc0, 0x12, 0xc0, 0x08,
    0x00, 0x16, 0x00, 0x13, 0xc0, 0x0d, 0xc0, 0x03, 0x00, 0x0a, 0xc0, 0x13,
    0xc0, 0x09, 0x00, 0x33, 0x00, 0x32, 0x00, 0x9a, 0x00, 0x99, 0x00, 0x45,
    0x00, 0x44, 0xc0, 0x0e, 0xc0, 0x04, 0x00, 0x2f, 0x00, 0x96, 0x00, 0x41,
    0x00, 0x07, 0x00, 0xff,       // Cipher Suites (32 suites)
    0x01,                         // Compression Methods Length: 1
    0x00,                         // Compression Methods (1 method)
    // Extensions Length [2 bytes]
    // Extensions [variable length]
}

var ClientHelloHandshakeHeartbeatExt = []byte{
    0x00, 0x0f, 0x00, 0x01, 0x01, // Extension: Heartbeat
}

func BuildClientHello(msgseq uint16, version int, extensions [][]byte) (packet, random []byte) {
	buf := bytes.Buffer{}
    var intermediate []byte
    var extlen uint16 = 0

    // build extension (last part of the packet)
    for i := range(extensions) {
        buf.Write(extensions[i])
        extlen += uint16(len(extensions[i]))
    }
    intermediate = buf.Bytes()
    fmt.Println(hex.Dump(intermediate))
    buf.Reset()

    // add extension length and extensions
    buf.Write(Uint16To2Bytes(extlen))
    buf.Write(intermediate)
    intermediate = buf.Bytes()
    fmt.Println(hex.Dump(intermediate))
    buf.Reset()

    // add version
    buf.Write(Uint16To2Bytes(uint16(version)))

    // generate and add timestamp
    epoch := uint32(time.Now().Unix())
    epochbuf := make([]byte, 4)
    binary.BigEndian.PutUint32(epochbuf, epoch)
    buf.Write(epochbuf)

    // generate and add random bytes
    randbuf := make([]byte, 28)
    rand.Read(randbuf)
    buf.Write(randbuf)

    // add last handshake part
    buf.Write(ClientHelloHandshakePart3)
    buf.Write(intermediate)
    intermediate = buf.Bytes()
    fmt.Println(hex.Dump(intermediate))
    FragmentLength := uint32(len(intermediate))
    buf.Reset()

    // add message sequence, fragment offset and fragment length
    buf.Write(Uint16To2Bytes(uint16(msgseq)))
    buf.Write(ClientHelloHandshakePart2)
    buf.Write(Uint32To3Bytes(FragmentLength))
    buf.Write(intermediate)
    intermediate = buf.Bytes()
    fmt.Println(hex.Dump(intermediate))
    Length := uint32(len(intermediate))
    buf.Reset()

    // add handshake type and length
    buf.Write(ClientHelloHandshakePart1)
    buf.Write(Uint32To3Bytes(Length))
    buf.Write(intermediate)

    return buf.Bytes(), randbuf
}

func Uint32To3Bytes(in uint32) []byte {
    if in > 255*255*255 {
        panic(fmt.Errorf("Unable to convert uint32 %d to a [3]byte", in))
    }
    buf := make([]byte, 4)
    binary.BigEndian.PutUint32(buf, in)
    return buf[1:4]
}

func Uint16To2Bytes(in uint16) []byte {
    buf := make([]byte, 2)
    binary.BigEndian.PutUint16(buf, in)
    return buf
}

func BuildDTLSRecord(ContentType, ProtocolVersion []byte, epoch uint16, seqnum uint64, fragment []byte) []byte {
	buf := bytes.Buffer{}

    // add ContentType
	buf.Write(ContentType)

    // add ProtocolVersion
	buf.Write(ProtocolVersion)

    // add epoch
    epochbuf := make([]byte, 2)
    binary.BigEndian.PutUint16(epochbuf, epoch)

    buf.Write(epochbuf)

    // add sequence number
    sequencebuf := make([]byte, 8)
    binary.BigEndian.PutUint64(sequencebuf, seqnum)

    buf.Write(sequencebuf[2:8])

    // calculate length of fragment and add length
    length64 := len(fragment)
    if length64 > 255*255-1 {
        panic("Fragment is too large.")
    }
    length16 := uint16(length64)
    lenbuf := make([]byte, 2)
    binary.BigEndian.PutUint16(lenbuf, length16)
	
    buf.Write(lenbuf)

    // add fragment
	buf.Write(fragment)

	return buf.Bytes()
}
