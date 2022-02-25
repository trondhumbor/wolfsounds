package wolfsounds

import (
	"bytes"
	"encoding/binary"
	"os"

	opl "github.com/trondhumbor/go-woody-opl"
)

type AudioHedEntry struct {
	Offset uint32
	Size   uint32
}

type PCSoundHead struct {
	Length   uint32
	Priority uint16
}

type PCSoundEntry struct {
	Header PCSoundHead
	Data   []byte
}

type VsWap struct {
	Header  VsWapHead
	Entries []VsWapEntry
}

type VsWapHead struct {
	ChunksInFile uint16
	SpriteStart  uint16
	SoundStart   uint16
}

type VsWapEntry struct {
	Offset uint32
	Size   uint16
	Data   []byte
}

type IMF struct {
	Length    uint16
	AdlibData []AdlibUnit
	ExtraData []byte
}

type AdlibUnit struct {
	AdlibRegister byte
	AdlibData     byte
	Delay         uint16
}

func ReadAudioHed(audioHedPath string) []AudioHedEntry {
	file, err := os.Open(audioHedPath)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	st, err := file.Stat()
	if err != nil {
		panic(err)
	}

	// file is a list of uint32le offsets, we can therefore just divide size by 4
	offsets := make([]uint32, st.Size()/4)
	binary.Read(file, binary.LittleEndian, offsets)

	var chunks []AudioHedEntry

	for i, off := range offsets[:len(offsets)-1] {
		chunks = append(chunks, AudioHedEntry{
			Offset: off,
			Size:   offsets[i+1] - offsets[i],
		})
	}
	return chunks
}

func ReadPCSounds(audioTPath string, chunks []AudioHedEntry) []PCSoundEntry {
	file, err := os.Open(audioTPath)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	var audio []PCSoundEntry

	for _, chunk := range chunks {
		file.Seek(int64(chunk.Offset), 0)

		head := PCSoundHead{}
		binary.Read(file, binary.LittleEndian, &head)

		data := make([]byte, head.Length)
		binary.Read(file, binary.LittleEndian, &data)

		audio = append(audio, PCSoundEntry{Header: head, Data: data})
	}

	return audio
}

func ReadIMF(audioTPath string, chunks []AudioHedEntry) []IMF {
	file, err := os.Open(audioTPath)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	var audio []IMF

	for _, chunk := range chunks {
		file.Seek(int64(chunk.Offset), 0)

		imf := IMF{}
		binary.Read(file, binary.LittleEndian, &imf.Length)

		data := make([]AdlibUnit, imf.Length/4) // sizeof(AdlibUnit) is 4 bytes
		binary.Read(file, binary.LittleEndian, &data)
		imf.AdlibData = data

		extraData := make([]byte, chunk.Size-uint32(imf.Length)-2)
		binary.Read(file, binary.LittleEndian, &extraData)
		imf.ExtraData = extraData

		audio = append(audio, imf)
	}

	return audio
}

func ReadVsWap(vsWapPath string) VsWap {
	file, err := os.Open(vsWapPath)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	head := VsWapHead{}
	binary.Read(file, binary.LittleEndian, &head)

	entries := make([]VsWapEntry, head.ChunksInFile)

	for i := 0; i < int(head.ChunksInFile); i++ {
		binary.Read(file, binary.LittleEndian, &entries[i].Offset)
	}

	for i := 0; i < int(head.ChunksInFile); i++ {
		binary.Read(file, binary.LittleEndian, &entries[i].Size)
	}

	for i := 0; i < int(head.ChunksInFile); i++ {
		file.Seek(int64(entries[i].Offset), 0)

		data := make([]byte, entries[i].Size)
		binary.Read(file, binary.LittleEndian, &data)
		entries[i].Data = data
	}

	return VsWap{Header: head, Entries: entries}
}

func ConvertAdlibSoundToPCM(sound IMF, pcmRate int, imfRate int) *bytes.Buffer {
	samplesPerEvent := pcmRate / imfRate

	buff := new(bytes.Buffer)

	o := *opl.NewOpl()
	o.Adlib_init(pcmRate)

	for _, e := range sound.AdlibData {
		o.Adlib_write(e.AdlibRegister, e.AdlibData)
		for i := 0; i < samplesPerEvent; i += 1 {
			binary.Write(buff, binary.LittleEndian, o.Adlib_getsample())
		}
		for i := 0; i < int(e.Delay)*samplesPerEvent; i += 1 {
			binary.Write(buff, binary.LittleEndian, o.Adlib_getsample())
		}
	}
	return buff
}

func ConvertPCSoundToPCM(sound PCSoundEntry, sampleRate uint32) *bytes.Buffer {
	const PC_BASE_TIMER = 1193181
	const PC_VOLUME = 20
	const PC_RATE = 140

	var sign int = -1
	var tone, phaseLength, phaseTic, samplesPerByte uint32 = 0, 0, 0, sampleRate / PC_RATE

	dst := new(bytes.Buffer)

	for _, b := range sound.Data {
		tone = uint32(b) * 60
		phaseLength = (sampleRate * tone) / (2 * PC_BASE_TIMER)
		for i := uint32(0); i < samplesPerByte; i += 1 {
			if tone != 0 {
				binary.Write(dst, binary.LittleEndian, uint8(128+sign*PC_VOLUME))
				if phaseTic >= phaseLength {
					sign = -sign
					phaseTic = 0
				} else {
					phaseTic += 1
				}
			} else {
				phaseTic = 0
				binary.Write(dst, binary.LittleEndian, uint8(128))
			}
		}
	}

	return dst
}

func WriteWavFile(outPath string, data *bytes.Buffer, sampleRate uint32, bitdepth uint32) {
	outFile, err := os.Create(outPath)

	if err != nil {
		panic(err)
	}

	defer outFile.Close()

	binary.Write(outFile, binary.BigEndian, &[]byte{82, 73, 70, 70}) // RIFF
	binary.Write(outFile, binary.LittleEndian, uint32(36+data.Len()))
	binary.Write(outFile, binary.BigEndian, &[]byte{87, 65, 86, 69})    // WAVE
	binary.Write(outFile, binary.BigEndian, &[]byte{102, 109, 116, 32}) // fmt
	binary.Write(outFile, binary.LittleEndian, uint32(16))
	binary.Write(outFile, binary.LittleEndian, uint16(1))
	binary.Write(outFile, binary.LittleEndian, uint16(1))
	binary.Write(outFile, binary.LittleEndian, sampleRate)
	binary.Write(outFile, binary.LittleEndian, sampleRate*bitdepth/8)
	binary.Write(outFile, binary.LittleEndian, uint16(bitdepth/8))
	binary.Write(outFile, binary.LittleEndian, uint16(bitdepth))

	binary.Write(outFile, binary.BigEndian, &[]byte{100, 97, 116, 97}) // data
	binary.Write(outFile, binary.LittleEndian, uint32(data.Len()))
	binary.Write(outFile, binary.LittleEndian, data.Bytes())
}
