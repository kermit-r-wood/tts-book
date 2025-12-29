package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// MergeWavFiles concatenates multiple WAV files into a single output file.
// It assumes all input files have the same format (sample rate, channels, bit depth).
func MergeWavFiles(inputs []string, outputPath string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no input files to merge")
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// 1. Read the header from the first file to use as a template
	firstFile, err := os.Open(inputs[0])
	if err != nil {
		return fmt.Errorf("failed to open first input file: %w", err)
	}
	defer firstFile.Close()

	header := make([]byte, 44) // Standard WAV header size
	if _, err := io.ReadFull(firstFile, header); err != nil {
		return fmt.Errorf("failed to read header from %s: %w", inputs[0], err)
	}

	// Write placeholder header to output (will update sizes later)
	if _, err := outFile.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	totalDataSize := uint32(0)

	// 2. Append data from all files (including the first one's body)
	// We need to re-read the first file's body, so we can just loop through inputs.
	// But for the first file, we already read the header.
	// Easier: Close firstFile and loop all fresh.
	firstFile.Close()

	for _, inputPath := range inputs {
		f, err := os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", inputPath, err)
		}

		// Skip header
		if _, err := f.Seek(44, 0); err != nil {
			f.Close()
			return fmt.Errorf("failed to seek in %s: %w", inputPath, err)
		}

		n, err := io.Copy(outFile, f)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to append data from %s: %w", inputPath, err)
		}
		totalDataSize += uint32(n)
	}

	// 3. Update File Size fields in the header
	// RIFF chunk size = 36 + totalDataSize
	// Data subchunk size = totalDataSize

	// Seek to RIFF chunk size (offset 4)
	if _, err := outFile.Seek(4, 0); err != nil {
		return fmt.Errorf("failed to seek to RIFF size: %w", err)
	}
	riffSize := totalDataSize + 36
	if err := binary.Write(outFile, binary.LittleEndian, riffSize); err != nil {
		return fmt.Errorf("failed to write RIFF size: %w", err)
	}

	// Seek to data subchunk size (offset 40)
	if _, err := outFile.Seek(40, 0); err != nil {
		return fmt.Errorf("failed to seek to data size: %w", err)
	}
	if err := binary.Write(outFile, binary.LittleEndian, totalDataSize); err != nil {
		return fmt.Errorf("failed to write data size: %w", err)
	}

	return nil
}
