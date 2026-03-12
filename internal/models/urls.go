package models

const (
	// ParakeetArchiveURL is the download URL for the Parakeet TDT 0.6B v3 INT8 model archive.
	ParakeetArchiveURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8.tar.bz2"

	// SileroVADURL is the download URL for the Silero VAD model.
	SileroVADURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/silero_vad.onnx"

	// SpeakerEmbeddingURL is the download URL for the 3D-Speaker embedding model.
	SpeakerEmbeddingURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-recongition-models/3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx"

	// ParakeetSubdir is the directory name inside the model archive after extraction.
	ParakeetSubdir = "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8"

	// SileroVADFile is the filename for the Silero VAD model.
	SileroVADFile = "silero_vad.onnx"

	// SpeakerEmbeddingFile is the filename for the speaker embedding model.
	SpeakerEmbeddingFile = "3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx"

	// PyannoteSegmentationURL is the download URL for the Pyannote speaker segmentation model archive.
	PyannoteSegmentationURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-segmentation-models/sherpa-onnx-pyannote-segmentation-3-0.tar.bz2"

	// PyannoteSegmentationSubdir is the directory name after extraction.
	PyannoteSegmentationSubdir = "sherpa-onnx-pyannote-segmentation-3-0"

	// PyannoteSegmentationFile is the model file inside the extraction directory.
	PyannoteSegmentationFile = "model.onnx"

	// Expected model files inside the Parakeet subdirectory.
	encoderFile = "encoder.int8.onnx"
	decoderFile = "decoder.int8.onnx"
	joinerFile  = "joiner.int8.onnx"
	tokensFile  = "tokens.txt"
)
