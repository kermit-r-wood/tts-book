package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// MergeWavFiles concatenates multiple WAV files into a single output file.
// It assumes all input files have the same format (sample rate, channels, bit depth).
func MergeWavFiles(inputs []string, outputPath string, silenceMs int) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no input files to merge")
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// 1. Read the header from the first file to extract format info
	firstFile, err := os.Open(inputs[0])
	if err != nil {
		return fmt.Errorf("failed to open first input file: %w", err)
	}
	defer firstFile.Close()

	// Read RIFF header (12 bytes: "RIFF" + size + "WAVE")
	riffHeader := make([]byte, 12)
	if _, err := io.ReadFull(firstFile, riffHeader); err != nil {
		return fmt.Errorf("failed to read RIFF header: %w", err)
	}

	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return fmt.Errorf("invalid WAV file format")
	}

	// Read chunks until we find "fmt " chunk
	var fmtChunk []byte
	var numChannels uint16
	var sampleRate uint32
	var bitsPerSample uint16

	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(firstFile, chunkHeader); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read chunk header: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			// Read fmt chunk data
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(firstFile, fmtData); err != nil {
				return fmt.Errorf("failed to read fmt chunk: %w", err)
			}
			fmtChunk = append(chunkHeader, fmtData...)

			// Parse format info
			numChannels = binary.LittleEndian.Uint16(fmtData[2:4])
			sampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])

			fmt.Printf("[Merger] WAV Format: channels=%d, sampleRate=%d, bitsPerSample=%d\n",
				numChannels, sampleRate, bitsPerSample)
			break
		} else {
			// Skip this chunk
			if _, err := firstFile.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return fmt.Errorf("failed to skip chunk %s: %w", chunkID, err)
			}
		}
	}

	if fmtChunk == nil {
		return fmt.Errorf("fmt chunk not found")
	}

	firstFile.Close()

	// Calculate silence buffer
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample/8)
	silenceBytes := (byteRate * uint32(silenceMs)) / 1000
	blockAlign := uint32(numChannels * (bitsPerSample / 8))
	if silenceBytes%blockAlign != 0 {
		silenceBytes += blockAlign - (silenceBytes % blockAlign)
	}
	silenceBuffer := make([]byte, silenceBytes)

	// Write RIFF header (placeholder, will update size later)
	outFile.Write([]byte("RIFF"))
	binary.Write(outFile, binary.LittleEndian, uint32(0)) // Placeholder
	outFile.Write([]byte("WAVE"))

	// Write fmt chunk
	outFile.Write(fmtChunk)

	// Write data chunk header (placeholder size)
	outFile.Write([]byte("data"))
	dataChunkSizePos, _ := outFile.Seek(0, io.SeekCurrent)
	binary.Write(outFile, binary.LittleEndian, uint32(0)) // Placeholder

	totalDataSize := uint32(0)

	// 2. Append audio data from all files
	for i, inputPath := range inputs {
		// Insert silence before every file except the first one
		if i > 0 && silenceMs > 0 {
			n, err := outFile.Write(silenceBuffer)
			if err != nil {
				return fmt.Errorf("failed to write silence: %w", err)
			}
			totalDataSize += uint32(n)
		}

		f, err := os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", inputPath, err)
		}

		// Skip to data chunk
		dataOffset, dataSize, err := findDataChunk(f)
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to find data chunk in %s: %w", inputPath, err)
		}

		if _, err := f.Seek(dataOffset, 0); err != nil {
			f.Close()
			return fmt.Errorf("failed to seek to data in %s: %w", inputPath, err)
		}

		// Copy only the audio data
		n, err := io.CopyN(outFile, f, int64(dataSize))
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to copy data from %s: %w", inputPath, err)
		}
		totalDataSize += uint32(n)
	}

	// 3. Update size fields
	// Update RIFF chunk size (fileSize - 8)
	if _, err := outFile.Seek(4, 0); err != nil {
		return fmt.Errorf("failed to seek to RIFF size: %w", err)
	}
	riffSize := uint32(len(fmtChunk)) + 8 + totalDataSize + 4 // fmt + "data" header + data + "WAVE"
	if err := binary.Write(outFile, binary.LittleEndian, riffSize); err != nil {
		return fmt.Errorf("failed to write RIFF size: %w", err)
	}

	// Update data chunk size
	if _, err := outFile.Seek(dataChunkSizePos, 0); err != nil {
		return fmt.Errorf("failed to seek to data size: %w", err)
	}
	if err := binary.Write(outFile, binary.LittleEndian, totalDataSize); err != nil {
		return fmt.Errorf("failed to write data size: %w", err)
	}

	fmt.Printf("[Merger] Successfully merged %d files, total data size: %d bytes\n", len(inputs), totalDataSize)
	return nil
}

// findDataChunk finds the data chunk in a WAV file and returns its offset and size
func findDataChunk(f *os.File) (int64, uint32, error) {
	// Skip RIFF header (12 bytes)
	if _, err := f.Seek(12, 0); err != nil {
		return 0, 0, err
	}

	for {
		chunkHeader := make([]byte, 8)
		currentPos, _ := f.Seek(0, io.SeekCurrent)

		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			if err == io.EOF {
				return 0, 0, fmt.Errorf("data chunk not found")
			}
			return 0, 0, err
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "data" {
			// Return offset (after chunk header) and size
			return currentPos + 8, chunkSize, nil
		}

		// Skip this chunk
		if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
			return 0, 0, err
		}
	}
}
