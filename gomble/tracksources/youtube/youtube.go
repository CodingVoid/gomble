package youtube

import (
    "encoding/json"
    "github.com/CodingVoid/gomble/gomble/audioformats"
    "github.com/CodingVoid/gomble/gomble/container/matroska"
    "github.com/CodingVoid/gomble/logger"
    "io"
    "io/ioutil"
    "net/http"
    "net/url"
    "errors"
    "fmt"
    "regexp"
    "runtime"
    "strconv"
    "strings"
)

type YoutubeVideo struct {
    matroskacont *matroska.Matroska
    blockOffset  int
    pcmbuff      []int16
    pcmbuffoff   int
    timeoffset   int
    // opus decoder
    dec         *audioformats.OpusDecoder
    doneReading bool

    title string
}

func NewYoutubeVideo(videoid string) (*YoutubeVideo, error) { // {{{
    //path = path + "&pbj=1&hl=en"
    path := "https://www.youtube.com/watch?v=" + videoid + "&pbj=1&hl=en";
    epath := "https://www.youtube.com/embed/" + videoid;
    var jsonbytes []byte
    // The following loop is for trying up to 3 times to download the youtube page and find a specific json string in it. You might not believe but it doesn't always work on the first try...
    var i int
    header := make(http.Header)
    header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) " + "Chrome/76.0.3809.100 Safari/537.36");
    header.Add("x-youtube-client-name", "1");
    header.Add("x-youtube-client-version", "2.20191008.04.01");
    header.Add("x-youtube-page-cl", "276511266");
    header.Add("x-youtube-page-label", "youtube.ytfe.desktop_20191024_3_RC0");
    header.Add("x-youtube-utc-offset", "0");
    header.Add("x-youtube-variants-checksum", "7a1198276cf2b23fc8321fac72aa876b");
    header.Add("accept-language", "en");
    for i = 0; i < 3 && len(jsonbytes) == 0; i++ {
        req, err := http.NewRequest("GET", path, nil)
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
        req.Header = header;
        resp, err := http.DefaultClient.Do(req);
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
        defer resp.Body.Close()

        jsonbytes, err = ioutil.ReadAll(resp.Body)
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
    }
    if i == 3 {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): Tried 3 times to Download and find ytplayer.config jsonstr. Done trying... ", file, line)
    }

    // DEBUG
    logger.Tracefile("youtube-output.json", jsonbytes)

    var f interface{}
    err := json.Unmarshal(jsonbytes, &f)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    m0 := f.([]interface{})

    var player_response map[string]interface{};
    var player map[string]interface{};
    var pstatus string;
    for _, elem := range m0 {
        melem := elem.(map[string]interface{});
        m00 := melem["playerResponse"];
        m01 := melem["player"];
        if m00 != nil {
            player_response = m00.(map[string]interface{});
            playabilityStatus := player_response["playabilityStatus"];
            if playabilityStatus == nil {
                return nil, errors.New("no playabilityStatus found");
            }
            pstatus = playabilityStatus.(map[string]interface{})["status"].(string);
        }
        if m01 != nil {
            player = m01.(map[string]interface{});
        }
    }
    logger.Debugf("status: " + pstatus + "\n")
    if pstatus != "OK" {
        return nil, errors.New("status was not ok. status: " + pstatus);
    }
    var basejs string
    if player == nil {
        // Search in embeded page for javascript url
        req, err := http.NewRequest("GET", epath, nil)
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
        req.Header = header;
        resp, err := http.DefaultClient.Do(req);
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
        body, err := ioutil.ReadAll(resp.Body);
        if err != nil {
            _, file, line, _ := runtime.Caller(0)
            return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
        }
        bodystr := string(body)
        defer resp.Body.Close();

        regex, err := regexp.Compile(`'WEB_PLAYER_CONTEXT_CONFIGS':(.*)}\);writeEmbed\(\);|'WEB_PLAYER_CONTEXT_CONFIGS':(.*)}\);yt.setConfig|\"WEB_PLAYER_CONTEXT_CONFIGS\":(.*)}\);writeEmbed\(\);|\"WEB_PLAYER_CONTEXT_CONFIGS\":(.*)}\);yt.setConfig|'PLAYER_CONFIG':(.*)}\);writeEmbed\(\);|'PLAYER_CONFIG':(.*)}\);yt.setConfig|\"PLAYER_CONFIG\":(.*)}\);writeEmbed\(\);|\"PLAYER_CONFIG\":(.*)}\);yt.setConfig`);
        jsonstrs := regex.FindStringSubmatch(bodystr);
        var jsonstr string
        for i,str := range jsonstrs {
            if str != "" && i > 0 {
                jsonstr = str;
                break;
            }
        }

        // DEBUG
        logger.Tracefile("youtube-embed.json", []byte(jsonstr))

        var baseEmbedPage interface{}
        err = json.Unmarshal([]byte(jsonstr), &baseEmbedPage);
        assets := baseEmbedPage.(map[string]interface{})["assets"];
        if (assets != nil) {
            basejs = assets.(map[string]interface{})["js"].(string);
        } else {
            basejs = baseEmbedPage.(map[string]interface{})["WEB_PLAYER_CONTEXT_CONFIG_ID_EMBEDDED_PLAYER"].(map[string]interface{})["jsUrl"].(string);
        }
        player = player_response;
    }
    if player["args"] != nil {
        if player["args"].(map[string]interface{})["adaptive_fmts"] != nil {
            //TODO loadTrackFormatsFromadaptive
            return nil, errors.New("TODO loadTrackFormatsFromadaptive");
        } else {
            //player["args"].(map[string]interface{})["player_response"].(string)
            return nil, errors.New("TODO load from player response unmarshal again");
        }
    }

    // New Format
    streamingData := player["streamingData"].(map[string]interface{})
    m6 := player["videoDetails"]
    //m7 := m6.(map[string]interface{})["isLiveContent"]
    //isLiveContent := m7.(bool)
    //logger.Debugf("isLiveContent: %t\n", isLiveContent)

    m8 := m6.(map[string]interface{})["lengthSeconds"]
    lengthSeconds := m8.(string)
    logger.Debugf("length: " + lengthSeconds + "\n")
    m9 := m6.(map[string]interface{})["title"]
    title := m9.(string)
    logger.Debugf("title: " + title + "\n")
    args0 := m6.(map[string]interface{})["author"]
    author := args0.(string)
    logger.Debugf("author: " + author + "\n")

    // find out media formats from json
    youtubeFormats := make([]youtubeFormat, 0)
    //m12 := args.(map[string]interface{})["adaptive_fmts"]

    //logger.Debugf("adaptive_fmts is nil, try another way to get formats\n")

    m13 := streamingData["formats"]
    formats := m13.([]interface{})
    logger.Debugf("%d formats:\n", len(formats))
    youtubeFormats = append(youtubeFormats[:], getYoutubeFormatList(formats)...)

    m14 := streamingData["adaptiveFormats"]
    adaptiveFormats := m14.([]interface{})
    logger.Debugf("%d adataptiveFormats:\n", len(adaptiveFormats))
    youtubeFormats = append(youtubeFormats[:], getYoutubeFormatList(adaptiveFormats)...)


    logger.Debugf("Find best format out of %d Formats\n", len(youtubeFormats))
    var bestFormat *youtubeFormat = findBestFormat(youtubeFormats)
    if bestFormat == nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): no appropriate format found: %w", file, line, err)
    }
    logger.Debugf("bestFormat: %v\n", *bestFormat)
    logger.Debugf("bestFormat CipherInfo: %v\n", bestFormat.cinfo)

    signatureUrl, err := resolveFormatUrl(bestFormat, basejs, false)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }

    // Now create a new youtube stream with our final url
    length, err := strconv.Atoi(bestFormat.contentLength)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    ystream, err := NewPersistentYoutubeStream(signatureUrl, length)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }

    odec, err := audioformats.NewOpusDecoder(48000, 2)

    matroskcont := matroska.NewMatroska(ystream)
    matroskcont.ReadHeader()

    y := &YoutubeVideo{
        matroskacont: matroskcont,
        dec:          odec,
        pcmbuff:      make([]int16, 0, 960),
        title:        title,
        doneReading:  false,
    }
    return y, nil
} // }}}

func resolveFormatUrl(format *youtubeFormat, playerScriptName string, skipResolving bool) (string, error) { // {{{
    if format.cinfo.skip || skipResolving {
        logger.Debugf("Skipping resolving of youtube url\n")
        return format.cinfo.url, nil
    }
    // get javascript of youtube player
    var playerjsurl string
    playerjsurl = "https://www.youtube.com" + playerScriptName
    logger.Debugf("playerjsurl: " + playerjsurl + "\n")

    resp, err := http.Get(playerjsurl)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("resolveFormatUrl(%s:%d): %w", file, line, err)
    }
    defer resp.Body.Close()

    // regex for javascript of youtube player to find out which cipher operations are applied on the signature (which we need to know to find out the final url of the actuall media/audio content)
    VARNAME := `[a-zA-Z_\$][a-zA-Z_0-9]*`
    BEFORE_ACCESS := `(?:\[\"|\.)`
    AFTER_ACCESS := `(?:\"\]|)`

    //functionPattern := "" +
    //"function(?: " + VARIABLE_PART + ")?\\(a\\)\\{" +
    //"a=a\\.split\\(\"\"\\);\\s*" +
    //"((?:(?:a=)?" + VARIABLE_PART + VARIABLE_PART_ACCESS + "\\(a,\\d+\\);)+)" +
    //"return a\\.join\\(\"\"\\)" +
    //"\\}"
    functionsPattern2 := `` +
    `function(?: ` + VARNAME + `)?\(a\)\{a=a\.split\(""\);\s*(` +
    `(?:` +
    `(?:a=)?` + VARNAME + `(?:\[\"|\.)` + VARNAME + `(?:\"\]|)\(a,\d+\);` +
    `)+` +
    `)return a\.join\(""\)\}`

    //actionsPattern := "" +
    //"var (" + VARIABLE_PART + ")=\\{((?:(?:" +
    //VARIABLE_PART_DEFINE + REVERSE_PART + "|" +
    //VARIABLE_PART_DEFINE + SLICE_PART + "|" +
    //VARIABLE_PART_DEFINE + SPLICE_PART + "|" +
    //VARIABLE_PART_DEFINE + SWAP_PART +
    //"),?\\n?)+)\\};"
    actionsPattern2 := `` +
    `var (` + VARNAME + `)=\{` +
    `(` +
    `(?:` +
    `(?:` +
    `\"?` + VARNAME + `\"?:function\(a\)\{(?:` +
    `return ` +
    `)?a\.reverse\(\)\}` +
    `|` +
    `\"?` + VARNAME + `\"?:function\(a,b\)\{return a\.slice\(b\)\}` +
    `|` +
    `\"?` + VARNAME + `\"?:function\(a,b\)\{a\.splice\(0,b\)\}` +
    `|` +
    `\"?` + VARNAME + `\"?:function\(a,b\)\{var c=a\[0\];a\[0\]=a\[b%a\.length\];a\[b(?:` +
    `%a.length|` +
    `)\]=c(?:` +
    `;return a` +
    `)?\}` +
    `),?\n?` +
    `)+` +
    `)` +
    `\};`

    PATTERN_PREFIX := `(?:^|,)\"?(` + VARNAME + `)\"?`

    REVERSE_PART := `:function\(a\)\{(?:return )?a\.reverse\(\)\}`
    SLICE_PART := `:function\(a,b\)\{return a\.slice\(b\)\}`
    SPLICE_PART := `:function\(a,b\)\{a\.splice\(0,b\)\}`
    SWAP_PART := `:function\(a,b\)\{` +
    `var c=a\[0\];a\[0\]=a\[b%a\.length\];a\[b(?:%a.length|)\]=c(?:;return a)?\}`

    reversePattern := "(?m)" + PATTERN_PREFIX + REVERSE_PART // (?m) = multiline mode
    slicePattern := "(?m)" + PATTERN_PREFIX + SLICE_PART
    splicePattern := "(?m)" + PATTERN_PREFIX + SPLICE_PART
    swapPattern := "(?m)" + PATTERN_PREFIX + SWAP_PART

    actions := regexp.MustCompile(actionsPattern2)

    byteResp, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    //err = ioutil.WriteFile("youtube.js", byteResp, 0777)
    //if err != nil {
    //  _, file, line, _ := runtime.Caller(0)
    //  return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    //}
    if !actions.MatchString(string(byteResp)) {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): Must find action functions from script: "+playerjsurl, file, line)
    }

    var actionBody string = actions.FindStringSubmatch(string(byteResp))[2]
    logger.Debugf("actionBody: " + actionBody)

    reverseKey := extractDollarEscapedFirstGroup(reversePattern, actionBody)
    logger.Debugf("reversePattern: " + reversePattern)
    slicePart := extractDollarEscapedFirstGroup(slicePattern, actionBody)
    logger.Debugf("slicePattern:   " + slicePattern)
    splicePart := extractDollarEscapedFirstGroup(splicePattern, actionBody)
    logger.Debugf("splicePattern:  " + splicePattern)
    swapKey := extractDollarEscapedFirstGroup(swapPattern, actionBody)
    logger.Debugf("swapPattern:    " + swapPattern)

    logger.Debugf("reverseKey: " + reverseKey)
    logger.Debugf("slicePart:  " + slicePart)
    logger.Debugf("splicePart: " + splicePart)
    logger.Debugf("swapKey:    " + swapKey)

    // \Q string \E = Pattern.quote = take 'string' as literal text (interpret nothing in it as a special meaning of regex) e.g.: ".*" matches regex ".*"
    all := make([]string, 0, 4)
    if reverseKey != "" {
        all = append(all, `\Q`+reverseKey+`\E`)
    }
    if slicePart != "" {
        all = append(all, `\Q`+slicePart+`\E`)
    }
    if splicePart != "" {
        all = append(all, `\Q`+splicePart+`\E`)
    }
    if swapKey != "" {
        all = append(all, `\Q`+swapKey+`\E`)
    }

    //all := []string {`\Q` + reverseKey + `\E`, `\Q` + slicePart + `\E`, `\Q` + splicePart + `\E`, `\Q` + swapKey + `\E`}

    // \Q string \E = Pattern.quote = take 'string' as literal text (interpret nothing in it as a special meaning of regex) e.g.: ".*" matches regex ".*"
    extractor := "(?:a=)?" + `\Q` + actions.FindStringSubmatch(string(byteResp))[1] + `\E` + BEFORE_ACCESS + "(" + strings.Join(all, "|") + ")" + AFTER_ACCESS + `\(a,(\d+)\)`

    logger.Debugf("extractor: " + extractor)
    fregex := regexp.MustCompile(functionsPattern2) // checks for functions
    if !fregex.MatchString(string(byteResp)) {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    extracterRegex := regexp.MustCompile(extractor) // extracts the cipher functions
    fun := fregex.FindStringSubmatch(string(byteResp))[1]
    logger.Debugf("fun: %s\n", fun)
    if !extracterRegex.MatchString(fun) {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }

    allGroupMatches := extracterRegex.FindAllStringSubmatch(string(byteResp), -1)
    operations := make([]CipherOperation, 0, 5)
    for i := 0; i < len(allGroupMatches); i++ {
        logger.Debugf("%d.%d: %s\n", i, 1, allGroupMatches[i][1])
        switch allGroupMatches[i][1] {
        case reverseKey:
            operations = append(operations, CipherOperation{optype: REVERSE, parameter: -1})
        case swapKey:
            param, err := strconv.ParseInt(allGroupMatches[i][2], 0, 64)
            if err != nil {
                logger.Fatalf("Failed to parse parameter")
            }
            operations = append(operations, CipherOperation{optype: SWAP, parameter: param})
        case splicePart:
            param, err := strconv.ParseInt(allGroupMatches[i][2], 0, 64)
            if err != nil {
                logger.Fatalf("Failed to parse parameter")
            }
            operations = append(operations, CipherOperation{optype: SLICE, parameter: param})
        case slicePart:
            param, err := strconv.ParseInt(allGroupMatches[i][2], 0, 64)
            if err != nil {
                logger.Fatalf("Failed to parse parameter")
            }
            operations = append(operations, CipherOperation{optype: SPLICE, parameter: param})
        }
    }

    if len(operations) == 0 {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): there are no operations (which can't be???): %w", file, line, err)
    }
    burl, err := url.Parse(format.cinfo.url)
    val := burl.Query()
    burl.RawQuery = val.Encode()
    val.Set("ratebypass", "yes")
    appliedSig, err := applyOperations(operations, format.cinfo.signature)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    //appliedSig := applyOperations(operations, "nUqnAZ-XtLr44YrRbKdWQFo7VAdVi8gc3U6gLmrMsGmsAEi=n0ovmI9nA8Gk0SpfSBNIzcEGRbw96-NeYpUDaJ_-yoOAhIgRwsLlPpJss")
    if len(appliedSig) == 0 {
        _, file, line, _ := runtime.Caller(0)
        return "", fmt.Errorf("NewYoutubeVideo(%s:%d): %w", file, line, err)
    }
    val.Set(format.cinfo.signaturekey, appliedSig)
    burl.RawQuery = val.Encode()
    return burl.String(), nil
} // }}}

// Returns a Mono PCMFrame with the given duration in milliseconds
func (y *YoutubeVideo) GetPCMFrame(duration int) ([]int16, error) { // {{{
    logger.Debugf("Called GetPCMFrame\n")
    neededSamples := 48 * duration // 48kHz * duration in ms
    // wait till we have the necessary pcm samples and buffer if possible
    for len(y.pcmbuff) < neededSamples && !y.doneReading {
        nextFrame, err := y.matroskacont.GetNextFrames(1)
        if err != nil {
            if err != io.EOF {
                _, file, line, _ := runtime.Caller(0)
                return nil, fmt.Errorf("GetPCMFrame(%s:%d): %w", file, line, err)
            }
            y.doneReading = true
        }
        logger.Debugf("len(nextFrame): %d\n", len(nextFrame))
        //samples, err := audioformats.GetPacketFrameSize(48000, nextFrame[0].Audiodata)
        //if err != nil {
        //  return nil, err
        //}
        //frameduration := 48000 / samples
        for i := 0; i < len(nextFrame); i++ {
            pcm, err := y.dec.Decode(nextFrame[i].Audiodata)
            if err != nil {
                _, file, line, _ := runtime.Caller(0)
                return nil, fmt.Errorf("GetPCMFrame(%s:%d): %w", file, line, err)
            }
            mono := make([]int16, len(pcm)/2)
            // Convert Stereo to Mono
            for j := 0; j < len(pcm)/2; j++ {
                mono[j] = (pcm[j*2]) //+ pcm[i*2+1]) / 2 // to take the average of both channels did really not work as expected... sounded awfull...
            }
            logger.Debugf("append mono length: %d to pcmbuff\n", len(mono))
            y.pcmbuff = append(y.pcmbuff, mono...)
        }
    }
    // if we still don't have enough samples, we probably have reached EOF and return the remaining samples
    if len(y.pcmbuff) < neededSamples {
        ret := make([]int16, len(y.pcmbuff))
        copy(ret, y.pcmbuff[:])
        logger.Debugf("GetPCMFrame EOF\n")
        return ret, io.EOF // return rest of pcmbuffer
    }
    logger.Debugf("Returned PCM length: %d pcmbuffer length: %d\n", neededSamples, len(y.pcmbuff))
    ret := make([]int16, neededSamples)
    copy(ret, y.pcmbuff[:neededSamples])
    y.pcmbuff = y.pcmbuff[neededSamples:]
    return ret, nil
} // }}}

func (y *YoutubeVideo) GetTitle() string {
    return y.title
}

type CipherOperation struct {
    optype    int
    parameter int64
}

const (
    SWAP int = iota
    REVERSE
    SLICE
    SPLICE
)

func getYoutubeFormatList(list []interface{}) []youtubeFormat { // {{{
    formats := make([]youtubeFormat, 0)
    for _, v := range list {
        var format youtubeFormat
        l := v.(map[string]interface{})

        if m1 := l["signatureCipher"]; m1 != nil { // try signatureCipher
            logger.Debugf("its signatureCipher\n")
            signatureCipher := m1.(string)
            cInfo, err := GetCipherInfoFromSignatureCipher(signatureCipher)
            if err != nil {
                logger.Debugf("Problem1.1\n")
                continue
            }
            cInfo.skip = false
            format.cinfo = cInfo
        } else if m1 := l["url"]; m1 != nil { // try url
            logger.Debugf("its url\n")
            url := m1.(string)
            cInfo, err := GetCipherInfoFromUrl(url)
            if err != nil {
                logger.Debugf("Problem1.2 " + err.Error() + "\n")
                continue
            }
            // cipher operations are skipped if there is a url instead of a signatureCipher
            cInfo.skip = true
            format.cinfo = cInfo
        }

        m2 := l["mimeType"] //TODO Do some kind of regex to differentiete between mimetype and codec
        if m2 == nil {
            logger.Debugf("No mimeType Information in the format infos\n")
            continue
        }
        format.mimeType = m2.(string)

        m3 := l["audioSampleRate"]
        if m3 == nil {
            logger.Debugf("No audioSampleRate Information in the format infos\n")
            continue
        }
        format.audioSampleRate = m3.(string)

        m4 := l["audioQuality"]
        if m4 == nil {
            logger.Debugf("No audioQuality Information in the format infos\n")
            continue
        }
        format.audioQuality = m4.(string)

        m5 := l["contentLength"]
        if m5 == nil {
            logger.Debugf("No contentLength Information in the format infos\n")
            continue
        }
        format.contentLength = m5.(string)

        m6 := l["audioChannels"]
        if m6 == nil {
            logger.Debugf("No audioChannels Information in the format infos\n")
            continue
        }
        format.audioChannels = m6.(float64)
        logger.Debugf("Added a format\n")
        formats = append(formats, format)
    }
    return formats
} // }}}

func applyOperations(operations []CipherOperation, text string) (string, error) { // {{{
    for _, op := range operations {
        switch op.optype {
        case SWAP:
            logger.Debugf("SWAP Operation\n")
            index := op.parameter % int64(len(text))
            tmp := text[0]
            // strings are immutable so we cannot assign at a specified index. so we have to do it like this
            text = string(text[index]) + text[1:]
            text = text[:index] + string(tmp) + text[index+1:]
        case REVERSE:
            logger.Debugf("REVERSE Operation\n")
            text = Reverse(text)
            //case SLICE:
            //logger.Debugf("SLICE Operation")
        case SPLICE, SLICE:
            logger.Debugf("SPLICE or SLICE Operation\n")
            text = text[op.parameter:] // remove the bytes before op.parameter
        default:
            _, file, line, _ := runtime.Caller(0)
            return "", fmt.Errorf("applyOperations(%s:%d): Invalid cipher operation", file, line)
        }
        logger.Debugf("text: " + text + "\n")
    }
    return text, nil
} // }}}

func Reverse(s string) string { // {{{
    n := len(s)
    runes := make([]rune, n)
    for _, rune := range s {
        n--
        runes[n] = rune
    }
    return string(runes[n:])
} // }}}

func extractDollarEscapedFirstGroup(pattern string, str string) string { // {{{
    regex := regexp.MustCompile(pattern)
    if regex.MatchString(str) {
        sm := regex.FindStringSubmatch(str)[1]
        return sm
    } else {
        return ""
    }
} // }}}

// returns value of given key in json data or an empty string if the key does not exists. So it does not distinguish between an empty value and no key found
//func get(root map[string]interface{}, key string, level int) string { // {{{
//  level++
//  for k, v := range root {
//      start:
//      switch vv := v.(type) {
//      case string:
//          // some json value strings are actually json objects, but with double quotes around them. But that doesn't seem to be standard JSON which is the reason why it didn't got parsed by go's unmarshal function of package json. To fix that we unmarshal again and replay the switch statement
//          if strings.HasPrefix(vv, `{`) {
//              var f interface{}
//              err := json.Unmarshal([]byte(vv), &f)
//              if err != nil {
//                  logger.Fatalf(err.Error())
//              }
//              m := f.(map[string]interface{})
//              v = m
//              logger.Fatalf("Got some invalid json")
//              goto start
//          }
//          logger.Printf("%d: %s: %s\n", level, k, vv)
//          if k == key {
//              return vv
//          }
//      case map[string]interface{}:
//          logger.Printf("%d: %s: one level down\n", level, k)
//          if v := get(vv, key, level); v != "" {
//              return v
//          }
//      }
//  }
//  return ""
//} // }}}

type cipherInfo struct {
    // url key=value - signaturekey=signature
    signaturekey string
    signature    string
    url          string
    // skip cipher operations?
    skip bool
}

// takes a url as string and returns a cipherinfo based on the json "signatureCipher"
func GetCipherInfoFromUrl(cipherUrl string) (*cipherInfo, error) { // {{{
    //query, err := url.ParseQuery(cipherUrl)
    //if err != nil {
    //  _, file, line, _ := runtime.Caller(0)
    //  return nil, fmt.Errorf("GetCipherInfoFromUrl(%s:%d): %w", file, line, err)
    //}
    //query := u.Query()

    var info cipherInfo
    info.url = cipherUrl
    //logger.Debugf("url: " + info.url + "\n")
    //info.signature = query.Get("sig")
    //if info.signature == "" {
    //  return nil, errors.New("no signature in cipherurl")
    //}
    //logger.Debugf("signature: " + info.signature + "\n")
    info.signaturekey = "signature"
    //logger.Debugf("signaturekey: " + info.signaturekey + "\n")
    return &info, nil
} // }}}

// takes a url as string and returns a cipherinfo based on the json "signatureCipher"
func GetCipherInfoFromSignatureCipher(cipherUrl string) (*cipherInfo, error) { // {{{
    query, err := url.ParseQuery(cipherUrl)
    if err != nil {
        _, file, line, _ := runtime.Caller(0)
        return nil, fmt.Errorf("GetCipherInfoFromUrl(%s:%d): %w", file, line, err)
    }
    //query := u.Query()

    var info cipherInfo
    //logger.Debugf("cipherUrl: " + cipherUrl + "\n")
    info.url = query.Get("url")
    if info.url == "" {
        return nil, errors.New("no url in cipherurl")
    }
    //logger.Debugf("url: " + info.url + "\n")
    info.signature = query.Get("s")
    if info.signature == "" {
        return nil, errors.New("no signature in cipherurl")
    }
    //logger.Debugf("signature: " + info.signature + "\n")
    info.signaturekey = query.Get("sp")
    if info.signaturekey == "" {
        return nil, errors.New("no signaturekey in cipherurl")
    }
    //logger.Debugf("signaturekey: " + info.signaturekey + "\n")
    return &info, nil
} // }}}

type youtubeFormat struct {
    cinfo           *cipherInfo
    contentLength   string
    audioSampleRate string
    audioQuality    string
    mimeType        string
    codec           string
    audioChannels   float64
}

// searches for the best format we can use (in this case webm and 48khz audio sample rate)
func findBestFormat(formats []youtubeFormat) *youtubeFormat { // {{{
    for _, f := range formats {
        logger.Debugf("Format: %v\n", f)
        if strings.Contains(f.mimeType, "webm") {
            if f.audioSampleRate == "48000" {
                //if f.audioQuality == "AUDIO_QUALITY_MEDIUM" {
                return &f
                //}
            }
        }
    }
    return nil
} // }}}


