package models

const (
	// ParakeetArchiveURL is the download URL for the Parakeet TDT 0.6B v2 FP16 model archive.
	ParakeetArchiveURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-fp16.tar.bz2"

	// SileroVADURL is the download URL for the Silero VAD model.
	SileroVADURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/silero_vad.onnx"

	// ParakeetSubdir is the directory name inside the model archive after extraction.
	ParakeetSubdir = "sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-fp16"

	// SileroVADFile is the filename for the Silero VAD model.
	SileroVADFile = "silero_vad.onnx"

	// Expected model files inside the Parakeet subdirectory.
	encoderFile = "encoder.fp16.onnx"
	decoderFile = "decoder.fp16.onnx"
	joinerFile  = "joiner.fp16.onnx"
	tokensFile  = "tokens.txt"
)

// parakeetModelFiles lists the required files inside the Parakeet model directory.
var parakeetModelFiles = []string{encoderFile, decoderFile, joinerFile, tokensFile}
