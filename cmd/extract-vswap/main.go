package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/trondhumbor/wolfsounds/internal/wolfsounds"
)

const pcmRate = 8000

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: wolfsounds-vswap vswap outdir")
		return
	}

	vsWapPath := os.Args[1]

	vsWap := wolfsounds.ReadVsWap(vsWapPath)
	for i, v := range vsWap.Entries[vsWap.Header.SoundStart:] {
		outPath := fmt.Sprintf("%s\\pcsound_%d.wav", os.Args[2], i)
		buff := new(bytes.Buffer)
		binary.Write(buff, binary.LittleEndian, &v.Data)

		wolfsounds.WriteWavFile(outPath, buff, pcmRate, 8)
	}
}
