package matroska

import "io"
import "time"
import "fmt"
import "github.com/CodingVoid/gomble/gomble/container/matroska/ebml"
import "github.com/CodingVoid/gomble/gomble/audioformats"
import "github.com/CodingVoid/gomble/logger"
import "runtime"

type Matroska struct {// {{{
	//clusters []Cluster
	//Metadata *File

	// Segment Information
	segmentInfo Info

	// Track Information
	track Track

	// selected Audiotrack
	satrack *TrackEntry

	// Clusters (Audio Data)
	clusters []Cluster

	// Decoder (it also represents the current EBML Element the decoder is at
	dec *ebml.Reader

	// blocks auf audiodata
	Blocks []Frame

	doneReadingMatroska bool
}// }}}

type Frame struct {
	ClusterTimecode time.Time
	RelativeBlockTime int16
	Audiodata []byte
}

func NewMatroska(reader io.Reader) *Matroska {
	m := &Matroska{}
	m.clusters = make([]Cluster, 0)
	m.Blocks = make([]Frame, 0)
	//m.Metadata = new(File)
	m.dec = ebml.NewReader(reader, &ebml.DecodeOptions{
		SkipDamaged: false,
	})
	return m
}

func (m *Matroska) ReadHeader() error {// {{{
	id, elem, err := m.dec.ReadElement()
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("ReadHeader(%s:%d): %w", file, line, err)
	}
	if id == Matroska_EBML {
		logger.Debugf("Found EBML Element")
		logger.Debugf("size: %d\n", elem.Len())
	}
	id, elem, err = m.dec.ReadElement()
	if id == Matroska_Segment {
		logger.Debugf("Found Segment Element")
		logger.Debugf("size: %d\n", elem.Len())
		// Parse Level 1 Elements
		cntNeeded := 0
		for {
			id, elema, err := elem.ReadElement()
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				return fmt.Errorf("ReadHeader(%s:%d): %w", file, line, err)
			}
			logger.Debugf("Level 1 next ELement id: 0x%08x\n", id)
			if id == Matroska_Info {
				// Parse Level 2 Elements
				err := elema.Decode(&m.segmentInfo)
				if err != nil {
					_, file, line, _ := runtime.Caller(0)
					return fmt.Errorf("ReadHeader(%s:%d): %w", file, line, err)
				}
				logger.Debugf("m.segmentInfo.Filename: %s\n", m.segmentInfo.Filename)
				logger.Debugf("m.segmentInfo.NextFilename: %s\n", m.segmentInfo.NextFilename)
				logger.Debugf("m.segmentInfo.Title: %s\n", m.segmentInfo.Title)
				logger.Debugf("m.segmentInfo.Duration: %f\n", m.segmentInfo.Duration)
				cntNeeded++
			}
			if id == Matroska_Tracks {
				// Parse Level 2 Elements
				err := elema.Decode(&m.track)
				if err != nil {
					_, file, line, _ := runtime.Caller(0)
					return fmt.Errorf("ReadHeader(%s:%d): %w", file, line, err)
				}
				logger.Debugf("Tracks:")
				for _, tr := range m.track.Entries {
					logger.Debugf("Tracknumber: %d\n", tr.Number)
					if tr.Audio != nil {
						logger.Debugf("Track is Audiotrack\n")
						logger.Debugf("tr.Audio.OutputSamplingFreq: %f\n", tr.Audio.OutputSamplingFreq)
						logger.Debugf("tr.Audio.SamplingFreq: %f\n", tr.Audio.SamplingFreq)
						logger.Debugf("tr.Audio.Channels: %d\n", tr.Audio.Channels)
						m.satrack = tr
					}
					if tr.Video != nil {
						logger.Debugf("Track is Videotrack\n")
					}
				}
				cntNeeded++
			}
			if id == Matroska_Cluster {
				// Parse Level 2 Elements
				err := elema.Decode(m.clusters)
				if err != nil {
					_, file, line, _ := runtime.Caller(0)
					return fmt.Errorf("ReadHeader(%s:%d): %w", file, line, err)
				}
				//TODO that's actually not an error, because it can happen that a Cluster Element comes before a Tracks or Info Element in a Matroska file (but it's unlikely)
				_, file, line, _ := runtime.Caller(0)
				return fmt.Errorf("ReadHeader(%s:%d): Got Cluster in the Header", file, line)
			}

			if cntNeeded == 2 {
				logger.Debugf("Got all needed for playback")
				m.dec = elem // save the point where the decoder was at
				break
			}
		}
	}
	return nil
	////TODO use some other STUFF that got decoded
}// }}}

// Read another Cluster (if the next element is a cluster)
func (m *Matroska) ReadContent() error {// {{{
	id, elem, err := m.dec.ReadElement()
	if err != nil {
		if err == io.EOF {
			logger.Debugf("End of Matroska File\n")
		}
		return err
	}
	// the ebml decoder cannot parse a Cluster, because it consists of Data not encoded in EBML. So we need to do it manually
	if id == Matroska_Cluster {
		//logger.Debugf("Found Cluster Element")
		//logger.Debugf("size: %d\n", elem.Len())
		var clustertimecode time.Time
		// looking for Cluster Timestamp
		for {
			id, elema, err := elem.ReadElement()
			if err != nil {
				_, file, line, _ := runtime.Caller(0)
				return fmt.Errorf("ReadContent(%s:%d): %w", file, line, err)
			}
			if id == Matroska_Timestamp {
				//logger.Debugf("Readed Timestamp successfully")
				clustertimecode, err = elema.ReadTime()
				if err != nil {
					_, file, line, _ := runtime.Caller(0)
					return fmt.Errorf("ReadContent(%s:%d): %w", file, line, err)
				}
				//logger.Debugf("Timestamp: " + clustertimecode.String())
				break
			}
		}
		// go through all Blocks in Cluster
		for {

			id, elemb, err := elem.ReadElement()
			if err != nil {
				if err == io.EOF {
					logger.Debugf("End of Cluster")
					break
				}
				_, file, line, _ := runtime.Caller(0)
				return fmt.Errorf("ReadContent(%s:%d): %w", file, line, err)
			}
			if id == Matroska_SimpleBlock {
				m.parseBlock(elemb, clustertimecode)
			} else if id == Matroska_BlockGroup {
				for {
					id, elemc, err := elemb.ReadElement()
					if err != nil {
						if err == io.EOF {
							logger.Debugf("End of Blockgroup")
							break
						}
						_, file, line, _ := runtime.Caller(0)
						return fmt.Errorf("ReadContent(%s:%d): %w", file, line, err)
					}
					if id == Matroska_Block {
						//logger.Debugf("Found Block Element")
						m.parseBlock(elemc, clustertimecode)
					}
				}

			} else {
				logger.Debugf("expected Block got: 0x%08x\n", id)
			}
		}
	} else {
		logger.Debugf("expected Cluster got: 0x%08x\n", id)
	}
	return nil
}// }}}

func (m *Matroska) parseBlock(reader *ebml.Reader, clusterTimecode time.Time) error {// {{{
	//logger.Debugf("size: %d\n", reader.Len())
	b := make([]byte, reader.Len())
	n, err := reader.Read(b[:])
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("parseBlock(%s:%d): %w", file, line, err)
	}
	if len(b) != n {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("parseBlock(%s:%d): Block len(b) != n", file, line)
	}
	//buf := bytes.NewBuffer(b[:])
	//blockReader := ebml.NewReader(buf, &ebml.DecodeOptions{
	//	SkipDamaged: false,
	//})
	blockReader := ebml.NewReaderBytes(b[:], &ebml.DecodeOptions{
		SkipDamaged: false,
	})
	//logger.Debugf("blockReader:  %d\n", blockReader.Len())
	//logger.Debugf("bufferlength: %d\n", buf.Len())

	//for _, by := range bcopy {
	//	fmt.Printf("%02x ", by)
	//}
	//fmt.Printf("\n")
	tracknum, err := blockReader.ReadVInt()
	logger.Debugf("Block of Tracknumber: %d\n", tracknum)
	//logger.Debugf("blockReader:  %d\n", blockReader.Len())
	num, err := blockReader.Next(2)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("parseBlock(%s:%d): %w", file, line, err)
	}
	blocktimecode := readInt16(num)
	//logger.Debugf("bufferlength: %d\n", buf.Len())
	timecode := clusterTimecode.Add(time.Duration(blocktimecode))
	//logger.Debugf("cluster-timecode: %d\n", clusterTimecode.Nanosecond())
	//logger.Debugf("block-timecode: %d\n", blocktimecode)
	logger.Debugf("timecode: %d\n", timecode.Nanosecond())
	flags, err := blockReader.Next(1)
	keyFrame := (flags[0] & 0x80) != 0
	logger.Debugf("keyframe: %b\n", keyFrame)
	laceType := (flags[0] & 0x06) >> 1
	if laceType != 0 {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("parseBlock(%s:%d): lacetype != 0", file, line)
	}
	//logger.Debugf("lacetype: %d\n", laceType)

	//logger.Debugf("blockReader: %d\n", blockReader.Len())
	remaining, err := blockReader.Next(int(blockReader.Len()))
	//logger.Debugf("blockReader: %d\n", blockReader.Len())
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		return fmt.Errorf("parseBlock(%s:%d): Error getting remaining bytes", file, line)
	}

	block := Frame{
		ClusterTimecode: clusterTimecode,
		RelativeBlockTime: blocktimecode,
		Audiodata: remaining,
	}
	m.Blocks = append(m.Blocks, block)
	return nil
}// }}}

//TODO need to return the Audiodata and not something specific for matroska like Frame (so I  can use the Interface MediaContainer properly)
// gets and removes the next matroska 'framecount' frames from the framebuffer
func (m *Matroska) GetNextFrames(framecount int) ([]Frame, error) {// {{{
	ret := make([]Frame, framecount)
	if !m.doneReadingMatroska {
		// Check if we have enough Blocks buffered
		for len(m.Blocks) <= framecount { // <= because we want to be able to return EOF properly
			logger.Debugf("Matroska not enough frames, need %d, have: %d\n", framecount, len(m.Blocks))
			if err := m.ReadContent(); err != nil {
				if err != io.EOF {
					_, file, line, _ := runtime.Caller(0)
					return nil, fmt.Errorf("GetNextFrames(%s:%d): %w", file, line, err)
				}
				m.doneReadingMatroska = true
				break //EOF
			}
		}
	}
	// if our buffer is still smaller or equals then framecount (which can only happen if EOF was reached in matroska), we just return the rest of the buffer
	if len(m.Blocks) <= framecount {
		logger.Debugf("Matroska still does not have enough frames in blocks-buffer\n")
		copy(ret[:], m.Blocks[:len(m.Blocks)])
		m.Blocks = nil //clear buffer for garbage collector
		return ret[:len(m.Blocks)], io.EOF
	}
	logger.Debugf("Matroska copy %d frames of %d to ret\n", framecount, len(m.Blocks))
	if n := copy(ret[:], m.Blocks[:framecount]); n < framecount {
		_, file, line, _ := runtime.Caller(0)
		return nil, fmt.Errorf("GetNextFrames(%s:%d): Could not return enough frames", file, line)
	}
	logger.Debugf("Matroska return frames: %d\n", len(ret))
	m.Blocks = m.Blocks[framecount:] // we do not want to use so much memory so we cut the parts we don't need anymore off
	return ret[:framecount], nil
}// }}}

func (m *Matroska) GetAudioCodec() audioformats.Codec {
	if m.satrack.CodecID == "A_OPUS" {
		return audioformats.CODEC_OPUS
	}
	return audioformats.CODEC_UNKNOWN
}

// read a signed int16 value from reader
func readInt16(b []byte) int16 {
	return ((int16(b[0]) << 8) | int16(b[1]))
}

// Matroska element types
const (// {{{
	// Level 0 Elements
	Matroska_EBML    = 0x1A45DFA3
	Matroska_Segment = 0x18538067

	// Header Elements
	Matroska_Version = 0x4286 // Level 1
	Matroska_EBMLMaxIDLength = 0x42F2 // Level 1
	Matroska_EBMLMaxSizeLength = 0x42F3 // Level 1

	// Meta Seek Information Elements
	Matroska_SeekHead = 0x114D9B74 // Level 1
	Matroska_Seek = 0x4DBB // Level 2
	Matroska_SeekID = 0x53AB // Level 3
	Matroska_SeekPosition = 0x53AC // Level 3

	// Segment Information
	Matroska_Info = 0x1549A966 // Level 1
	Matroska_SegmentUID = 0x73A4 // Level 2
	Matroska_SegmentFilename = 0x7384 // Level 2
	Matroska_PrevUID = 0x3CB923 // Level 2
	Matroska_PrevFilename = 0x3C83AB // Level 2
	Matroska_NextUID		= 0x3EB923
	Matroska_NextFilename	= 0x3E83BB
	Matroska_SegmentFamily	= 0x4444
	Matroska_ChapterTranslate	= 0x6924
	Matroska_ChapterTranslateEditionUID	= 0x69FC
	Matroska_ChapterTranslateCodec	= 0x69BF
	Matroska_ChapterTranslateID	= 0x69A5
	Matroska_TimestampScale	= 0x2AD7B1
	Matroska_Duration	= 0x4489
	Matroska_DateUTC		= 0x4461
	Matroska_Title	= 0x7BA9
	Matroska_MuxingApp	= 0x4D80
	Matroska_WritingApp	= 0x5741

	// Cluster
	Matroska_Cluster		= 0x1F43B675
	Matroska_Timestamp	= 0xE7
	Matroska_SilentTracks	= 0x5854
	Matroska_SilentTrackNumber	= 0x58D7
	Matroska_Position	= 0xA7
	Matroska_PrevSize	= 0xAB
	Matroska_SimpleBlock		= 0xA3
	Matroska_BlockGroup	= 0xA0
	Matroska_Block	= 0xA1
	Matroska_BlockVirtual	= 0xA2
	Matroska_BlockAdditions	= 0x75A1
	Matroska_BlockMore	= 0xA6
	Matroska_BlockAddID	= 0xEE
	Matroska_BlockAdditional		= 0xA5
	Matroska_BlockDuration	= 0x9B
	Matroska_ReferencePriority	= 0xFA
	Matroska_ReferenceBlock	= 0xFB
	Matroska_ReferenceVirtual	= 0xFD
	Matroska_CodecState	= 0xA4
	Matroska_DiscardPadding	= 0x75A2
	Matroska_Slices	= 0x8E
	Matroska_TimeSlice	= 0xE8
	Matroska_LaceNumber	= 0xCC
	Matroska_FrameNumber		= 0xCD
	Matroska_BlockAdditionID		= 0xCB
	Matroska_Delay	= 0xCE
	Matroska_SliceDuration	= 0xCF
	Matroska_ReferenceFrame	= 0xC8
	Matroska_ReferenceOffset		= 0xC9
	Matroska_ReferenceTimestamp	= 0xCA
	Matroska_EncryptedBlock	= 0xAF

	// Track
	Matroska_Tracks	= 0x1654AE6B
	Matroska_TrackEntry	= 0xAE
	Matroska_TrackNumber		= 0xD7
	Matroska_TrackUID	= 0x73C5
	Matroska_TrackType	= 0x83
	Matroska_FlagEnabled		= 0xB9
	Matroska_FlagDefault		= 0x88
	Matroska_FlagForced	= 0x55AA
	Matroska_FlagLacing	= 0x9C
	Matroska_MinCache	= 0x6DE7
	Matroska_MaxCache	= 0x6DF8
	Matroska_DefaultDuration		= 0x23E383
	Matroska_DefaultDecodedFieldDuration		= 0x234E7A
	Matroska_TrackTimestampScale		= 0x23314F
	Matroska_TrackOffset		= 0x537F
	Matroska_MaxBlockAdditionID	= 0x55EE
	Matroska_Name	= 0x536E
	Matroska_Language	= 0x22B59C
	Matroska_LanguageIETF	= 0x22B59D
	Matroska_CodecID		= 0x86
	Matroska_CodecPrivate	= 0x63A2
	Matroska_CodecName	= 0x258688
	Matroska_AttachmentLink	= 0x7446
	Matroska_CodecSettings	= 0x3A9697
	Matroska_CodecInfoURL	= 0x3B4040
	Matroska_CodecDownloadURL	= 0x26B240
	Matroska_CodecDecodeAll	= 0xAA
	Matroska_TrackOverlay	= 0x6FAB
	Matroska_CodecDelay	= 0x56AA
	Matroska_SeekPreRoll		= 0x56BB
	Matroska_TrackTranslate	= 0x6624
	Matroska_TrackTranslateEditionUID	= 0x66FC
	Matroska_TrackTranslateCodec		= 0x66BF
	Matroska_TrackTranslateTrackID	= 0x66A5
	Matroska_Video	= 0xE0
	Matroska_FlagInterlaced	= 0x9A
	Matroska_FieldOrder	= 0x9D
	Matroska_StereoMode	= 0x53B8
	Matroska_AlphaMode	= 0x53C0
	Matroska_OldStereoMode	= 0x53B9
	Matroska_PixelWidth	= 0xB0
	Matroska_PixelHeight		= 0xBA
	Matroska_PixelCropBottom		= 0x54AA
	Matroska_PixelCropTop	= 0x54BB
	Matroska_PixelCropLeft	= 0x54CC
	Matroska_PixelCropRight	= 0x54DD
	Matroska_DisplayWidth	= 0x54B0
	Matroska_DisplayHeight	= 0x54BA
	Matroska_DisplayUnit		= 0x54B2
	Matroska_AspectRatioType		= 0x54B3
	Matroska_ColourSpace		= 0x2EB524
	Matroska_GammaValue	= 0x2FB523
	Matroska_FrameRate	= 0x2383E3
	Matroska_Colour	= 0x55B0
	Matroska_MatrixCoefficients	= 0x55B1
	Matroska_BitsPerChannel	= 0x55B2
	Matroska_ChromaSubsamplingHorz	= 0x55B3
	Matroska_ChromaSubsamplingVert	= 0x55B4
	Matroska_CbSubsamplingHorz	= 0x55B5
	Matroska_CbSubsamplingVert	= 0x55B6
	Matroska_ChromaSitingHorz	= 0x55B7
	Matroska_ChromaSitingVert	= 0x55B8
	Matroska_Range	= 0x55B9
	Matroska_TransferCharacteristics		= 0x55BA
	Matroska_Primaries	= 0x55BB
	Matroska_MaxCLL	= 0x55BC
	Matroska_MaxFALL		= 0x55BD
	Matroska_MasteringMetadata	= 0x55D0
	Matroska_PrimaryRChromaticityX	= 0x55D1
	Matroska_PrimaryRChromaticityY	= 0x55D2
	Matroska_PrimaryGChromaticityX	= 0x55D3
	Matroska_PrimaryGChromaticityY	= 0x55D4
	Matroska_PrimaryBChromaticityX	= 0x55D5
	Matroska_PrimaryBChromaticityY	= 0x55D6
	Matroska_WhitePointChromaticityX		= 0x55D7
	Matroska_WhitePointChromaticityY		= 0x55D8
	Matroska_LuminanceMax	= 0x55D9
	Matroska_LuminanceMin	= 0x55DA
	Matroska_Projection	= 0x7670
	Matroska_ProjectionType	= 0x7671
	Matroska_ProjectionPrivate	= 0x7672
	Matroska_ProjectionPoseYaw	= 0x7673
	Matroska_ProjectionPosePitch		= 0x7674
	Matroska_ProjectionPoseRoll	= 0x7675
	Matroska_Audio	= 0xE1
	Matroska_SamplingFrequency	= 0xB5
	Matroska_OutputSamplingFrequency		= 0x78B5
	Matroska_Channels	= 0x9F
	Matroska_ChannelPositions	= 0x7D7B
	Matroska_BitDepth	= 0x6264
	Matroska_TrackOperation	= 0xE2
	Matroska_TrackCombinePlanes	= 0xE3
	Matroska_TrackPlane	= 0xE4
	Matroska_TrackPlaneUID	= 0xE5
	Matroska_TrackPlaneType	= 0xE6
	Matroska_TrackJoinBlocks		= 0xE9
	Matroska_TrackJoinUID	= 0xED
	Matroska_TrickTrackUID	= 0xC0
	Matroska_TrickTrackSegmentUID	= 0xC1
	Matroska_TrickTrackFlag	= 0xC6
	Matroska_TrickMasterTrackUID		= 0xC7
	Matroska_TrickMasterTrackSegmentUID	= 0xC4
	Matroska_ContentEncodings	= 0x6D80
	Matroska_ContentEncoding		= 0x6240
	Matroska_ContentEncodingOrder	= 0x5031
	Matroska_ContentEncodingScope	= 0x5032
	Matroska_ContentEncodingType		= 0x5033
	Matroska_ContentCompression	= 0x5034
	Matroska_ContentCompAlgo		= 0x4254
	Matroska_ContentCompSettings		= 0x4255
	Matroska_ContentEncryption	= 0x5035
	Matroska_ContentEncAlgo	= 0x47E1
	Matroska_ContentEncKeyID		= 0x47E2
	Matroska_ContentEncAESSettings	= 0x47E7
	Matroska_AESSettingsCipherMode	= 0x47E8
	Matroska_ContentSignature	= 0x47E3
	Matroska_ContentSigKeyID		= 0x47E4
	Matroska_ContentSigAlgo	= 0x47E5
	Matroska_ContentSigHashAlgo	= 0x47E6

	// Cueing Data
	Matroska_Cues	= 0x1C53BB6B
	Matroska_CuePoint	= 0xBB
	Matroska_CueTime		= 0xB3
	Matroska_CueTrackPositions	= 0xB7
	Matroska_CueTrack	= 0xF7
	Matroska_CueClusterPosition	= 0xF1
	Matroska_CueRelativePosition		= 0xF0
	Matroska_CueDuration		= 0xB2
	Matroska_CueBlockNumber	= 0x5378
	Matroska_CueCodecState	= 0xEA
	Matroska_CueReference	= 0xDB
	Matroska_CueRefTime	= 0x96
	Matroska_CueRefCluster	= 0x97
	Matroska_CueRefNumber	= 0x535F
	Matroska_CueRefCodecState	= 0xEB
	Matroska_Attachments		= 0x1941A469
	Matroska_AttachedFile	= 0x61A7
	Matroska_FileDescription		= 0x467E
	Matroska_FileName	= 0x466E
	Matroska_FileMimeType	= 0x4660
	Matroska_FileData	= 0x465C
	Matroska_FileUID		= 0x46AE
	Matroska_FileReferral	= 0x4675
	Matroska_FileUsedStartTime	= 0x4661
	Matroska_FileUsedEndTime		= 0x4662

	// Chapters
	Matroska_Chapters	= 0x1043A770
	Matroska_EditionEntry	= 0x45B9
	Matroska_EditionUID	= 0x45BC
	Matroska_EditionFlagHidden	= 0x45BD
	Matroska_EditionFlagDefault	= 0x45DB
	Matroska_EditionFlagOrdered	= 0x45DD
	Matroska_ChapterAtom		= 0xB6
	Matroska_ChapterUID	= 0x73C4
	Matroska_ChapterStringUID	= 0x5654
	Matroska_ChapterTimeStart	= 0x91
	Matroska_ChapterTimeEnd	= 0x92
	Matroska_ChapterFlagHidden	= 0x98
	Matroska_ChapterFlagEnabled	= 0x4598
	Matroska_ChapterSegmentUID	= 0x6E67
	Matroska_ChapterSegmentEditionUID	= 0x6EBC
	Matroska_ChapterPhysicalEquiv	= 0x63C3
	Matroska_ChapterTrack	= 0x8F
	Matroska_ChapterTrackNumber	= 0x89
	Matroska_ChapterDisplay	= 0x80
	Matroska_ChapString	= 0x85
	Matroska_ChapLanguage	= 0x437C
	Matroska_ChapLanguageIETF	= 0x437D
	Matroska_ChapCountry		= 0x437E
	Matroska_ChapProcess		= 0x6944
	Matroska_ChapProcessCodecID	= 0x6955
	Matroska_ChapProcessPrivate	= 0x450D
	Matroska_ChapProcessCommand	= 0x6911
	Matroska_ChapProcessTime		= 0x6922
	Matroska_ChapProcessData		= 0x6933
	Matroska_Tags	= 0x1254C367
	Matroska_Tag		= 0x7373
	Matroska_Targets		= 0x63C0
	Matroska_TargetTypeValue		= 0x68CA
	Matroska_TargetType	= 0x63CA
	Matroska_TagTrackUID		= 0x63C5
	Matroska_TagEditionUID	= 0x63C9
	Matroska_TagChapterUID	= 0x63C4
	Matroska_TagAttachmentUID	= 0x63C6
	Matroska_SimpleTag	= 0x67C8
	Matroska_TagName		= 0x45A3
	Matroska_TagLanguage		= 0x447A
	Matroska_TagLanguageIETF		= 0x447B
	Matroska_TagDefault	= 0x4484
	Matroska_TagString	= 0x4487
	Matroska_TagBinary	= 0x4485
)// }}}

//TODO from here on it's just structs for Matroska Elements (need to remove the unused ones)

// The EBML is a top level element contains a description of the file type.
type EBML struct {
	Version            int    `ebml:"4286,1"`
	ReadVersion        int    `ebml:"42F7,1"`
	MaxIDLength        int    `ebml:"42F2,4"`
	MaxSizeLength      int    `ebml:"42F3,8"`
	DocType            string `ebml:"4282,matroska"`
	DocTypeVersion     int    `ebml:"4287,1"`
	DocTypeReadVersion int    `ebml:"4285,1"`
}

// ID is a binary EBML element identifier.
type ID uint32

// SegmentID is a randomly generated unique 128bit identifier of Segment/SegmentFamily.
type SegmentID []byte

// EditionID is a unique identifier of Edition.
type EditionID []byte

type Position int64
type Time int64
type Duration int64

// Segment is the Root Element that contains all other Top-Level Elements.
type Segment struct {
	SeekHead    []*SeekHead   `ebml:"114D9B74,omitempty" json:",omitempty"`
	Info        []*Info       `ebml:"1549A966" json:",omitempty"`
	Cluster     []*Cluster    `ebml:"1F43B675,omitempty" json:",omitempty"`
	Tracks      []*Track      `ebml:"1654AE6B,omitempty" json:",omitempty"`
	Cues        []*CuePoint   `ebml:"1C53BB6B>BB,omitempty" json:",omitempty"`
	Attachments []*Attachment `ebml:"1941A469>61A7"`
	Chapters    []*Edition    `ebml:"1043A770>45B9"`
	Tags        []*Tag        `ebml:"1254C367>7373"`
}

// SeekHead contains the position of other Top-Level Elements.
type SeekHead struct {
	Seeks []*Seek `ebml:"4DBB"`
}

// Seek contains a single seek entry to an EBML Element.
type Seek struct {
	ID       ID       `ebml:"53AB"`
	Position Position `ebml:"53AC"`
}

// Info contains miscellaneous general information and statistics on the file.
type Info struct {
	ID               SegmentID           `ebml:"73A4,omitempty" json:",omitempty"`
	Filename         string              `ebml:"7384,omitempty" json:",omitempty"`
	PrevID           SegmentID           `ebml:"3CB923,omitempty" json:",omitempty"`
	PrevFilename     string              `ebml:"3C83AB,omitempty" json:",omitempty"`
	NextID           SegmentID           `ebml:"3EB923,omitempty" json:",omitempty"`
	NextFilename     string              `ebml:"3E83BB,omitempty" json:",omitempty"`
	SegmentFamily    SegmentID           `ebml:"4444,omitempty" json:",omitempty"`
	ChapterTranslate []*ChapterTranslate `ebml:"6924,omitempty" json:",omitempty"`
	TimecodeScale    time.Duration       `ebml:"2AD7B1,1000000"`
	Duration         float64             `ebml:"4489,omitempty" json:",omitempty"`
	Date             time.Time           `ebml:"4461,omitempty" json:",omitempty"`
	Title            string              `ebml:"7BA9,omitempty" json:",omitempty"`
	MuxingApp        string              `ebml:"4D80"`
	WritingApp       string              `ebml:"5741"`
}

// ChapterTranslate contains tuple of corresponding ID used by chapter codecs to represent a Segment.
type ChapterTranslate struct {
	EditionIDs []EditionID  `ebml:"69FC,omitempty" json:",omitempty"`
	Codec      ChapterCodec `ebml:"69BF"`
	ID         TranslateID  `ebml:"69A5"`
}

type TranslateID []byte
type ChapterCodec uint8

const (
	ChapterCodecMatroska ChapterCodec = iota
	ChapterCodecDVD
)

// Cluster is a Top-Level Element containing the Block structure.
type Cluster struct {
	Timecode     Time          `ebml:"E7"`
	SilentTracks []TrackNumber `ebml:"5854>58D7,omitempty" json:",omitempty"`
	Position     Position      `ebml:"A7,omitempty" json:",omitempty"`
	PrevSize     int64         `ebml:"AB,omitempty" json:",omitempty"`
	SimpleBlock  []*Block      `ebml:"A3,omitempty" json:",omitempty"`
	BlockGroup   []*BlockGroup `ebml:"A0,omitempty" json:",omitempty"`
}

type ClusterID uint64

// Block contains the actual data to be rendered and a timestamp.
type Block struct {
	TrackNumber TrackNumber
	Timecode    int16
	Flags       uint8
	Frames      int
	//Data []byte
}

const (
	LacingNone uint8 = iota
	LacingXiph
	LacingFixedSize
	LacingEBML
)

// BlockGroup contains a single Block and a relative information.
type BlockGroup struct {
	Block             *Block           `ebml:"A1" json:",omitempty"`
	Additions         []*BlockAddition `ebml:"75A1>A6,omitempty" json:",omitempty"`
	Duration          Duration         `ebml:"9B,omitempty" json:",omitempty"`
	ReferencePriority int64            `ebml:"FA"`
	ReferenceBlock    []Time           `ebml:"FB,omitempty" json:",omitempty"`
	CodecState        []byte           `ebml:"A4,omitempty" json:",omitempty"`
	DiscardPadding    time.Duration    `ebml:"75A2,omitempty" json:",omitempty"`
	Slices            []*TimeSlice     `ebml:"8E>E8,omitempty" json:",omitempty"`
}

type TimeSlice struct {
	LaceNumber int64 `ebml:"CC"`
}

// BlockAdd contains additional blocks to complete the main one.
type BlockAddition struct {
	ID   BlockAdditionID `ebml:"EE,1"`
	Data []byte          `ebml:"A5"`
}

type BlockAdditionID uint64

// Track is a Top-Level Element of information with track description.
type Track struct {
	Entries []*TrackEntry `ebml:"AE"`
}

// TrackEntry describes a track with all Elements.
type TrackEntry struct {
	Number                      TrackNumber        `ebml:"D7"`
	ID                          TrackID            `ebml:"73C5"`
	Type                        TrackType          `ebml:"83"`
	Enabled                     bool               `ebml:"B9,true"`
	Default                     bool               `ebml:"88,true"`
	Forced                      bool               `ebml:"55AA"`
	Lacing                      bool               `ebml:"9C,true"`
	MinCache                    int                `ebml:"6DE7"`
	MaxCache                    int                `ebml:"6DF8,omitempty" json:",omitempty"`
	DefaultDuration             time.Duration      `ebml:"23E383,omitempty" json:",omitempty"`
	DefaultDecodedFieldDuration time.Duration      `ebml:"234E7A,omitempty" json:",omitempty"`
	MaxBlockAdditionID          BlockAdditionID    `ebml:"55EE"`
	Name                        string             `ebml:"536E,omitempty" json:",omitempty"`
	Language                    string             `ebml:"22B59C,eng,omitempty" json:",omitempty"`
	CodecID                     string             `ebml:"86"`
	CodecPrivate                []byte             `ebml:"63A2,omitempty" json:",omitempty"`
	CodecName                   string             `ebml:"258688,omitempty" json:",omitempty"`
	AttachmentLink              AttachmentID       `ebml:"7446,omitempty" json:",omitempty"`
	CodecDecodeAll              bool               `ebml:"AA,true"`
	TrackOverlay                []TrackNumber      `ebml:"6FAB,omitempty" json:",omitempty"`
	CodecDelay                  time.Duration      `ebml:"56AA,omitempty" json:",omitempty"`
	SeekPreRoll                 time.Duration      `ebml:"56BB"`
	TrackTranslate              []*TrackTranslate  `ebml:"6624,omitempty" json:",omitempty"`
	Video                       *VideoTrack        `ebml:"E0,omitempty" json:",omitempty"`
	Audio                       *AudioTrack        `ebml:"E1,omitempty" json:",omitempty"`
	TrackOperation              *TrackOperation    `ebml:"E2,omitempty" json:",omitempty"`
	ContentEncodings            []*ContentEncoding `ebml:"6D80>6240,omitempty" json:",omitempty"`
}

type TrackID uint64
type TrackNumber int
type AttachmentID uint8
type TrackType uint8

const (
	TrackTypeVideo    TrackType = 0x01
	TrackTypeAudio    TrackType = 0x02
	TrackTypeComplex  TrackType = 0x03
	TrackTypeLogo     TrackType = 0x10
	TrackTypeSubtitle TrackType = 0x11
	TrackTypeButton   TrackType = 0x12
	TrackTypeControl  TrackType = 0x20
)

// TrackTranslate describes a track identification for the given Chapter Codec.
type TrackTranslate struct {
	EditionIDs       []EditionID  `ebml:"66FC,omitempty" json:",omitempty"`
	Codec            ChapterCodec `ebml:"66BF"`
	TranslateTrackID `ebml:"66A5"`
}

type TranslateTrackID []byte

// VideoTrack contains information that is specific for video tracks.
type VideoTrack struct {
	Interlaced      InterlaceType   `ebml:"9A"`
	FieldOrder      FieldOrder      `ebml:"9D,2"`
	StereoMode      StereoMode      `ebml:"53B8,omitempty" json:"stereoMode,omitempty"`
	AlphaMode       *AlphaMode      `ebml:"53C0,omitempty" json:"alphaMode,omitempty"`
	Width           int             `ebml:"B0"`
	Height          int             `ebml:"BA"`
	CropBottom      int             `ebml:"54AA,omitempty" json:",omitempty"`
	CropTop         int             `ebml:"54BB,omitempty" json:",omitempty"`
	CropLeft        int             `ebml:"54CC,omitempty" json:",omitempty"`
	CropRight       int             `ebml:"54DD,omitempty" json:",omitempty"`
	DisplayWidth    int             `ebml:"54B0,omitempty" json:",omitempty"`
	DisplayHeight   int             `ebml:"54BA,omitempty" json:",omitempty"`
	DisplayUnit     DisplayUnit     `ebml:"54B2,omitempty" json:",omitempty"`
	AspectRatioType AspectRatioType `ebml:"54B3,omitempty" json:",omitempty"`
	ColourSpace     uint32          `ebml:"2EB524,omitempty" json:",omitempty"`
	Colour          *Colour         `ebml:"55B0,omitempty" json:",omitempty"`
}

type InterlaceType uint8

// InterlaceTypes
const (
	InterlaceTypeInterlaced  InterlaceType = 1
	InterlaceTypeProgressive InterlaceType = 2
)

type FieldOrder uint8

// FieldOrders
const (
	FieldOrderProgressive           FieldOrder = 0
	FieldOrderTop                   FieldOrder = 1
	FieldOrderUndetermined          FieldOrder = 2
	FieldOrderBottom                FieldOrder = 6
	FieldOrderDisplayBottomStoreTop FieldOrder = 9
	FieldOrderDisplayTopStoreBottom FieldOrder = 14
)

type StereoMode uint8

// StereoModes
const (
	StereoModeMono StereoMode = iota
	StereoModeHorizontalLeft
	StereoModeVerticalRight
	StereoModeVerticalLeft
	StereoModeCheckboardRight
	StereoModeCheckboardLeft
	StereoModeInterleavedRight
	StereoModeInterleavedLeft
	StereoModeColumnInterleavedRight
	StereoModeAnaglyphCyanRed
	StereoModeHorizontalRight
	StereoModeAnaglyphGreenMagenta
	StereoModeLacedLeft
	StereoModeLacedRight
)

type AlphaMode struct {
}

type DisplayUnit uint8

// DisplayUnits
const (
	DisplayUnitPixels DisplayUnit = iota
	DisplayUnitCentimeters
	DisplayUnitInches
	DisplayUnitAspectRatio
)

type AspectRatioType uint8

// AspectRatioTypes
const (
	AspectRatioFreeResizing AspectRatioType = iota
	AspectRatioKeep
	AspectRatioFixed
)

// Colour describes the colour format settings.
type Colour struct {
	MatrixCoefficients      MatrixCoefficients      `ebml:"55B1,2,omitempty" json:",omitempty"`
	BitsPerChannel          int                     `ebml:"55B2,omitempty" json:",omitempty"`
	ChromaSubsamplingHorz   int                     `ebml:"55B3,omitempty" json:",omitempty"`
	ChromaSubsamplingVert   int                     `ebml:"55B4,omitempty" json:",omitempty"`
	CbSubsamplingHorz       int                     `ebml:"55B5,omitempty" json:",omitempty"`
	CbSubsamplingVert       int                     `ebml:"55B6,omitempty" json:",omitempty"`
	ChromaSitingHorz        ChromaSiting            `ebml:"55B7,omitempty" json:",omitempty"`
	ChromaSitingVert        ChromaSiting            `ebml:"55B8,omitempty" json:",omitempty"`
	ColourRange             ColourRange             `ebml:"55B9,omitempty" json:",omitempty"`
	TransferCharacteristics TransferCharacteristics `ebml:"55BA,omitempty" json:",omitempty"`
	Primaries               Primaries               `ebml:"55BB,2,omitempty" json:",omitempty"`
	MaxCLL                  int64                   `ebml:"55BC,omitempty" json:",omitempty"`
	MaxFALL                 int64                   `ebml:"55BD,omitempty" json:",omitempty"`
	MasteringMetadata       *MasteringMetadata      `ebml:"55D0"`
}

// MatrixCoefficients, see Table 4 of ISO/IEC 23001-8:2013/DCOR1
type MatrixCoefficients uint8

// TransferCharacteristics, see Table 3 of ISO/IEC 23001-8:2013/DCOR1
type TransferCharacteristics uint8

// Primaries, see Table 2 of ISO/IEC 23001-8:2013/DCOR1
type Primaries uint8

type ChromaSiting uint8

// ChromaSitings
const (
	ChromaSitingUnspecified ChromaSiting = iota
	ChromaSitingCollocated
	ChromaSitingHalf
)

type ColourRange uint8

// ColourRange
const (
	ColourRangeUnspecified ColourRange = iota
	ColourRangeBroadcast
	ColourRangeFull
	ColourRangeDefined
)

// MasteringMetadata represents SMPTE 2086 mastering data.
type MasteringMetadata struct {
	PrimaryRChromaX   float64 `ebml:"55D1,omitempty" json:",omitempty"`
	PrimaryRChromaY   float64 `ebml:"55D2,omitempty" json:",omitempty"`
	PrimaryGChromaX   float64 `ebml:"55D3,omitempty" json:",omitempty"`
	PrimaryGChromaY   float64 `ebml:"55D4,omitempty" json:",omitempty"`
	PrimaryBChromaX   float64 `ebml:"55D5,omitempty" json:",omitempty"`
	PrimaryBChromaY   float64 `ebml:"55D6,omitempty" json:",omitempty"`
	WhitePointChromaX float64 `ebml:"55D7,omitempty" json:",omitempty"`
	WhitePointChromaY float64 `ebml:"55D8,omitempty" json:",omitempty"`
	LuminanceMax      float64 `ebml:"55D9,omitempty" json:",omitempty"`
	LuminanceMin      float64 `ebml:"55DA,omitempty" json:",omitempty"`
}

// AudioTrack contains information that is specific for audio tracks.
type AudioTrack struct {
	SamplingFreq       float64 `ebml:"B5,8000"`
	OutputSamplingFreq float64 `ebml:"78B5,omitempty" json:",omitempty"`
	Channels           int     `ebml:"9F,1"`
	BitDepth           int     `ebml:"6264,omitempty" json:",omitempty"`
}

// TrackOperation describes an operation that needs to be applied on tracks
// to create the virtual track.
type TrackOperation struct {
	CombinePlanes []*TrackPlane `ebml:"E3>E4,omitempty" json:",omitempty"`
	JoinBlocks    []TrackID     `ebml:"E9>ED,omitempty" json:",omitempty"`
}

// TrackPlane contains a video plane track that need to be combined to create this track.
type TrackPlane struct {
	ID   TrackID   `ebml:"E5"`
	Type PlaneType `ebml:"E6"`
}

type PlaneType uint8

// PlaneTypes
const (
	PlaneTypeLeft PlaneType = iota
	PlaneTypeRight
	PlaneTypeBackground
)

// ContentEncoding contains settings for several content encoding mechanisms
// like compression or encryption.
type ContentEncoding struct {
	Order       int           `ebml:"5031"`
	Scope       EncodingScope `ebml:"5032,1"`
	Type        EncodingType  `ebml:"5033"`
	Compression *Compression  `ebml:"5034,omitempty" json:",omitempty"`
	Encryption  *Encryption   `ebml:"5035,omitempty" json:",omitempty"`
}

type EncodingScope uint8

// EncodingScopes
const (
	EncodingScopeAll     EncodingScope = 1
	EncodingScopePrivate EncodingScope = 2
	EncodingScopeNext    EncodingScope = 4
)

type EncodingType uint8

const (
	EncodingTypeCompression EncodingType = iota
	EncodingTypeEncryption
)

// Compression describes the compression used.
type Compression struct {
	Algo     CompressionAlgo `ebml:"4254"`
	Settings []byte          `ebml:"4255,omitempty" json:",omitempty"`
}

type CompressionAlgo uint8

const (
	CompressionAlgoZlib            CompressionAlgo = 0
	CompressionAlgoHeaderStripping CompressionAlgo = 3
)

// Encryption describes the encryption used.
type Encryption struct {
	Algo         uint8  `ebml:"47E1,omitempty" json:",omitempty"`
	KeyID        []byte `ebml:"47E2,omitempty" json:",omitempty"`
	Signature    []byte `ebml:"47E3,omitempty" json:",omitempty"`
	SignKeyID    []byte `ebml:"47E4,omitempty" json:",omitempty"`
	SignAlgo     uint8  `ebml:"47E5,omitempty" json:",omitempty"`
	SignHashAlgo uint8  `ebml:"47E6,omitempty" json:",omitempty"`
}

// CuePoint contains all information relative to a seek point in the Segment.
type CuePoint struct {
	Time           Time                `ebml:"B3"`
	TrackPositions []*CueTrackPosition `ebml:"B7"`
}

// CueTrackPosition contains positions for different tracks corresponding to the timestamp.
type CueTrackPosition struct {
	Track            TrackNumber `ebml:"F7"`
	ClusterPosition  Position    `ebml:"F1"`
	RelativePosition Position    `ebml:"F0,omitempty" json:",omitempty"`
	Duration         Duration    `ebml:"B2,omitempty" json:",omitempty"`
	BlockNumber      int         `ebml:"5378,1,omitempty" json:",omitempty"`
	CodecState       Position    `ebml:"EA,omitempty" json:",omitempty"`
	References       []Time      `ebml:"DB>96,omitempty" json:",omitempty"`
}

// Attachment describes attached files.
type Attachment struct {
	ID          AttachmentID `ebml:"46AE"`
	Description string       `ebml:"467E,omitempty" json:",omitempty"`
	Name        string       `ebml:"466E"`
	MimeType    string       `ebml:"4660"`
	Data        []byte       `ebml:"465C"`
}

// Edition contains all information about a Segment edition.
type Edition struct {
	ID      EditionID      `ebml:"45BC,omitempty" json:",omitempty"`
	Hidden  bool           `ebml:"45BD"`
	Default bool           `ebml:"45DB"`
	Ordered bool           `ebml:"45DD,omitempty" json:",omitempty"`
	Atoms   []*ChapterAtom `ebml:"B6"`
}

// ChapterAtom contains the atom information to use as the chapter atom.
type ChapterAtom struct {
	ID            ChapterID         `ebml:"73C4"`
	StringID      string            `ebml:"5654,omitempty" json:",omitempty"`
	TimeStart     Time              `ebml:"91"`
	TimeEnd       Time              `ebml:"92,omitempty" json:",omitempty"`
	Hidden        bool              `ebml:"98"`
	Enabled       bool              `ebml:"4598,true"`
	SegmentID     SegmentID         `ebml:"6E67,omitempty" json:",omitempty"`
	EditionID     EditionID         `ebml:"6EBC,omitempty" json:",omitempty"`
	PhysicalEquiv int               `ebml:"63C3,omitempty" json:",omitempty"`
	Tracks        []TrackID         `ebml:"8F>89,omitempty" json:",omitempty"`
	Displays      []*ChapterDisplay `ebml:"80,omitempty" json:",omitempty"`
	Processes     []*ChapterProcess `ebml:"6944,omitempty" json:",omitempty"`
}

type ChapterID uint64

// ChapterDisplay contains all possible strings to use for the chapter display.
type ChapterDisplay struct {
	String   string `ebml:"85"`
	Language string `ebml:"437C,eng"`                         // See ISO-639-2
	Country  string `ebml:"437E,omitempty" json:",omitempty"` // See IANA ccTLDs
}

// ChapterProcess describes the atom processing commands.
type ChapterProcess struct {
	CodecID ChapterCodec      `ebml:"6955"`
	Private []byte            `ebml:"450D,omitempty" json:",omitempty"`
	Command []*ChapterCommand `ebml:"6911,omitempty" json:",omitempty"`
}

// ChapterCommand contains all the commands associated to the atom.
type ChapterCommand struct {
	Time Time   `ebml:"6922"`
	Data []byte `ebml:"6933"`
}

// Tag contains Elements specific to Tracks/Chapters.
type Tag struct {
	Targets    []*Target    `ebml:"63C0"`
	SimpleTags []*SimpleTag `ebml:"67C8"`
}

// Target contains all IDs where the specified meta data apply.
type Target struct {
	TypeValue     int            `ebml:"68CA,50,omitempty" json:",omitempty"`
	Type          string         `ebml:"63CA,omitempty" json:",omitempty"`
	TrackIDs      []TrackID      `ebml:"63C5,omitempty" json:",omitempty"`
	EditionIDs    []EditionID    `ebml:"63C9,omitempty" json:",omitempty"`
	ChapterIDs    []ChapterID    `ebml:"63C4,omitempty" json:",omitempty"`
	AttachmentIDs []AttachmentID `ebml:"63C6,omitempty" json:",omitempty"`
}

// SimpleTag contains general information about the target.
type SimpleTag struct {
	Name     string `ebml:"45A3"`
	Language string `ebml:"447A,und"`
	Default  bool   `ebml:"4484,true"`
	String   string `ebml:"4487,omitempty" json:",omitempty"`
	Binary   []byte `ebml:"4485,omitempty" json:",omitempty"`
}

func NewSimpleTag(name, text string) *SimpleTag {
	return &SimpleTag{
		Name:     name,
		String:   text,
		Language: "und",
		Default:  true,
	}
}
