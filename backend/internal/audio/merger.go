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

// NormalizeAudio performs peak normalization on a WAV file using pure Go.
// It currently only supports 16-bit PCM (linear) WAV files.
// It will normalize the audio to -1.0 dB (approx 90% peak).
func NormalizeAudio(inputPath, outputPath string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	// Parse Header
	riffHeader := make([]byte, 12)
	if _, err := io.ReadFull(f, riffHeader); err != nil {
		return fmt.Errorf("failed to read RIFF: %w", err)
	}

	// Read chunks to find fmt and data
	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var bitsPerSample uint16
	var dataOffset int64
	var dataSize uint32

	// We need to rewind to parse correctly if we re-use logic, but let's just scan manually
	// to ensure we capture AudioFormat
	chunkHeader := make([]byte, 8)
	for {
		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read chunk header failed: %w", err)
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, fmtData); err != nil {
				return fmt.Errorf("read fmt data failed: %w", err)
			}
			audioFormat = binary.LittleEndian.Uint16(fmtData[0:2])
			numChannels = binary.LittleEndian.Uint16(fmtData[2:4])
			sampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])
		} else if chunkID == "data" {
			dataOffset, _ = f.Seek(0, io.SeekCurrent)
			dataSize = chunkSize
			break // Data found, stop scanning (assuming data is last or we just need it)
		} else {
			f.Seek(int64(chunkSize), io.SeekCurrent)
		}
	}

	// Validation
	if audioFormat != 1 {
		return fmt.Errorf("unsupported audio format %d (only PCM=1 is supported)", audioFormat)
	}
	if bitsPerSample != 16 {
		return fmt.Errorf("unsupported bit depth %d (only 16-bit is supported)", bitsPerSample)
	}

	// Pass 1: Find Peak
	if _, err := f.Seek(dataOffset, 0); err != nil {
		return err
	}

	maxPeak := int16(0)
	buf := make([]byte, 4096) // Read in chunks

	bytesRead := uint32(0)
	for bytesRead < dataSize {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// Process buffer
		limit := n
		if uint32(limit) > dataSize-bytesRead {
			limit = int(dataSize - bytesRead)
		}

		for i := 0; i < limit; i += 2 {
			val := int16(binary.LittleEndian.Uint16(buf[i : i+2]))
			if val < 0 {
				val = -val
			}
			if val > maxPeak {
				maxPeak = val
			}
		}
		bytesRead += uint32(n)
	}

	if maxPeak == 0 {
		// Silent file, just copy
		f.Seek(0, 0)
		out, _ := os.Create(outputPath)
		io.Copy(out, f)
		out.Close()
		return nil
	}

	// Calculate Gain
	// Target: -1.0 dB ~ 0.891 * 32767 = 29195
	targetPeak := 29195.0
	gain := targetPeak / float64(maxPeak)

	if gain < 1.0 {
		// Don't reduce volume, only boost. Or should we?
		// Normalization usually implies matching target.
		// If it's already louder than -1dB, we should reduce it to avoid clipping/consistency.
	}

	fmt.Printf("[Normalizer] MaxPeak: %d, Gain: %.4f\n", maxPeak, gain)

	// Pass 2: Apply Gain and Write
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Parse original file headers again to copy them exactly?
	// Simpler: Just reconstruct the header.
	// RIFF Header
	out.Write([]byte("RIFF"))
	binary.Write(out, binary.LittleEndian, uint32(36+dataSize))
	out.Write([]byte("WAVE"))

	// fmt chunk
	out.Write([]byte("fmt "))
	binary.Write(out, binary.LittleEndian, uint32(16)) // Subchunk1Size for PCM
	binary.Write(out, binary.LittleEndian, uint16(1))  // AudioFormat 1
	binary.Write(out, binary.LittleEndian, numChannels)
	binary.Write(out, binary.LittleEndian, sampleRate)
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample/8)
	binary.Write(out, binary.LittleEndian, byteRate)
	blockAlign := uint16(numChannels * (bitsPerSample / 8))
	binary.Write(out, binary.LittleEndian, blockAlign)
	binary.Write(out, binary.LittleEndian, bitsPerSample)

	// data chunk header
	out.Write([]byte("data"))
	binary.Write(out, binary.LittleEndian, dataSize)

	// Write data
	f.Seek(dataOffset, 0)
	bytesRead = 0

	writeBuf := make([]byte, 4096)

	for bytesRead < dataSize {
		n, err := f.Read(buf) // reuse reading buf
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		limit := n
		if uint32(limit) > dataSize-bytesRead {
			limit = int(dataSize - bytesRead)
		}

		for i := 0; i < limit; i += 2 {
			val := int16(binary.LittleEndian.Uint16(buf[i : i+2]))

			newVal := int32(float64(val) * gain)
			// Clamp
			if newVal > 32767 {
				newVal = 32767
			} else if newVal < -32768 {
				newVal = -32768
			}

			binary.LittleEndian.PutUint16(writeBuf[i:i+2], uint16(int16(newVal)))
		}

		out.Write(writeBuf[:limit])
		bytesRead += uint32(n)
	}

	return nil
}

// MergeAndNormalize works like MergeWavFiles but optionally applies normalization
func MergeAndNormalize(inputs []string, outputPath string, silenceMs int, normalize bool) error {
	if !normalize {
		return MergeWavFiles(inputs, outputPath, silenceMs)
	}

	// Merge to a temporary file first
	tempMerged := outputPath + ".tmp.wav"
	// Ensure temp file is cleaned up
	defer os.Remove(tempMerged)

	if err := MergeWavFiles(inputs, tempMerged, silenceMs); err != nil {
		return err
	}

	// Normalize
	if err := NormalizeAudio(tempMerged, outputPath); err != nil {
		fmt.Printf("Normalization failed: %v. Using un-normalized audio.\n", err)
		// If normalization fails (e.g. wrong format), just move the temp file
		os.Rename(tempMerged, outputPath)
		// Consider returning nil error so process doesn't stop?
		// But let's log it.
		return nil
	}

	return nil
}
