package tracksources

//import "testing"
//
//import "os"
//
//var opusfilepath string = "/home/max/Downloads/music.opus"
//
//var oggFile OggOpusFile
//func openfile(t *testing.T) {
//	loggFile, err := Gop_open_file(opusfilepath)
//	if err != nil {
//		t.Errorf("Error: %s\n", err)
//	}
//	oggFile = loggFile
//}
//
//var totalSamplesPerChannel int64
//func getPcmTotal(t *testing.T) {
//	t.Run("open file", openfile)
//
//	ltotalSamplesPerChannel, err := Gop_pcm_total(oggFile, -1)
//	if err != nil {
//		t.Errorf("Error: %s\n", err.Error())
//	}
//	totalSamplesPerChannel = ltotalSamplesPerChannel
//}
//
//var pcmBuffer []int16
//func readStereo(t *testing.T) {
//	t.Run("get total samples per channel", getPcmTotal)
//	//frame_size := 960
//	//sample_rate := 48000
//	//channels := 2
//	//bitrate := 64000
//	//max_frame_size := 9*960
//	//max_packet_size := 3*1276
//
//	var pcmOffset int = 0
//
//	var pcm [120*48*2]int16 // 120ms of data at 48kHz per channel (11520 values total)
//	pcmBuffer = make([]int16, ((960*2) - totalSamplesPerChannel*2 % (960*2)) + totalSamplesPerChannel*2)
//	for {
//		readedSamples, err := Gop_read_stereo(oggFile, pcm[:], uint32(len(pcm)))
//		if err != nil {
//			t.Errorf("Error: %s\n", err.Error())
//		}
//		copy(pcmBuffer[pcmOffset:], pcm[:readedSamples*2])
//		pcmOffset += (readedSamples*2)
//		if pcmOffset == int(totalSamplesPerChannel*2) {
//			// readedSamples == 0 would work too, but this checks if our query before worked out too
//			t.Log("Success End-Of-File")
//			break;
//		}
//		if (pcmOffset > len(pcmBuffer)) {
//			t.Fatal("Buffer to small\n")
//		}
//	}
//	writeInt16InFile(t, "in.pcm", 0, pcmBuffer[:pcmOffset])
//}
//
//
//func TestEncodeOpus(t *testing.T) {
//	t.Run("read Stereo", readStereo)
//
//	outputFile, err := os.Create("testout.opus") // Removes the file if it already exists
//	outputFile.Close()
//
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//
//	opusenc, err := Gopus_encoder_create(48000, 2, -1) //TODO -1 is not right
//	if err != nil {
//		t.Fatalf("Gopus_encoder_create error: %s\n", err.Error())
//	}
//	encOffset := 0
//	for i := 0; i < len(pcmBuffer); i += 960*2 {
//		var opusPayload [4000]byte
//		//panic: runtime error: slice bounds out of range [:12414720] with capacity 12414336
//		encoded, err := Gopus_encode(opusenc, pcmBuffer[i:i+960*2], 960, opusPayload[:], len(opusPayload))
//		if err != nil {
//			t.Fatalf("Error while encoding opus: %s\n", err.Error())
//		}
//		writeInFile(t, "testout.opus", int64(encOffset), opusPayload[:encoded])
//		encOffset += encoded
//	}
//	t.Log("Now start testout.opus with media player to confirm success\n")
//}
//
//// write buffer in file of path with offset
//func writeInFile(t *testing.T, path string, offset int64, buffer []byte) {
//	file, err := os.OpenFile(path, os.O_RDWR, 0666)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	if _, err := file.WriteAt(buffer[:], offset); err != nil {
//		t.Fatal(err.Error())
//	}
//	file.Close()
//}
//
//// write int16 buffer in file of path with offset
//func writeInt16InFile(t *testing.T, path string, offset int64, buffer []int16) {
//	file, err := os.Create(path)
//	if err != nil {
//		t.Fatal(err.Error())
//	}
//	bufferByte := make([]byte, len(buffer)*2)
//	// Convert to little Endian ordering
//	for i := 0; i < len(buffer); i++ {
//		bufferByte[2*i] = byte(buffer[i] & 0xFF)
//		bufferByte[2*i+1] = byte((buffer[i] >> 8) & 0xFF)
//	}
//
//	if _, err := file.WriteAt(bufferByte[:], offset); err != nil {
//		t.Fatal(err.Error())
//	}
//}
