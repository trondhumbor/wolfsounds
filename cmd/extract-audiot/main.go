package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/trondhumbor/wolfsounds/internal/wolfsounds"
)

type SoundOffsets struct {
	StartPCSounds    int
	StartAdlibSounds int
	StartDigiSounds  int
	StartMusic       int
}

const (
	pcmRate = 44100
	imfRate = 700
)

func main() {
	gameOffsets := map[string]SoundOffsets{
		"WL1": {0, 69, 138, 207},
		"WL6": {0, 87, 174, 261},
		"SOD": {0, 81, 162, 243},
	}

	if len(os.Args) != 4 {
		fmt.Println("usage: wolfsounds-audiot audiohed audiot outdir")
		return
	}

	audioHed := os.Args[1]
	audioT := os.Args[2]

	var offset SoundOffsets
	if off, present := gameOffsets[filepath.Ext(audioHed)[1:]]; present {
		offset = off
	} else {
		fmt.Println("Couldn't determine offsets. Unrecognized file extension")
		return
	}

	audioTEntries := wolfsounds.ReadAudioHed(audioHed)
	for i, v := range wolfsounds.ReadPCSounds(audioT, audioTEntries[offset.StartPCSounds:offset.StartAdlibSounds]) {
		outPath := fmt.Sprintf("%s\\pcsound_%d.wav", os.Args[3], i)

		pcm := wolfsounds.ConvertPCSoundToPCM(v, pcmRate)
		wolfsounds.WriteWavFile(outPath, pcm, pcmRate, 8)
	}

	for i, v := range wolfsounds.ReadIMF(audioT, audioTEntries[offset.StartMusic:]) {
		outPath := fmt.Sprintf("%s\\adlib_music_%d.wav", os.Args[3], i)

		pcm := wolfsounds.ConvertAdlibSoundToPCM(v, pcmRate, imfRate)
		wolfsounds.WriteWavFile(outPath, pcm, pcmRate, 16)
	}
}
