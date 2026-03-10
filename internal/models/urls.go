package models

const (
	// ParakeetArchiveURL is the download URL for the Parakeet TDT 0.6B v3 INT8 model archive.
	ParakeetArchiveURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8.tar.bz2"

	// SileroVADURL is the download URL for the Silero VAD model.
	SileroVADURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/silero_vad.onnx"

	// ParakeetSubdir is the directory name inside the model archive after extraction.
	ParakeetSubdir = "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8"

	// SileroVADFile is the filename for the Silero VAD model.
	SileroVADFile = "silero_vad.onnx"

	// Expected model files inside the Parakeet subdirectory.
	encoderFile = "encoder.int8.onnx"
	decoderFile = "decoder.int8.onnx"
	joinerFile  = "joiner.int8.onnx"
	tokensFile  = "tokens.txt"
)
