package gomble

import (
    "fmt"
    "os"

    "github.com/CodingVoid/gomble/logger"
)

// write int16 buffer in file of path with offset
func WriteInt16InFile(path string, buffer []int16) { // {{{
    file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0777)
    if err != nil {
        logger.Fatalf(err.Error())
    }
    bufferByte := make([]byte, len(buffer)*2)
    // Convert to little Endian ordering
    for i := 0; i < len(buffer); i++ {
        bufferByte[2*i] = byte(buffer[i] & 0xFF)
        bufferByte[2*i+1] = byte((buffer[i] >> 8) & 0xFF)
    }

    if _, err := file.WriteAt(bufferByte[:], 0); err != nil {
        logger.Fatalf(err.Error())
    }
} // }}}

// write buffer in file of path with offset
func writeInFile(path string, offset int64, buffer []byte) { // {{{
    file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        logger.Fatalf(err.Error())
    }
    n := 0
    if n, err = file.WriteAt(buffer[:], offset); err != nil {
        logger.Fatalf(err.Error())
    }
    logger.Debugf("Wrote %d bytes\n", n)
    file.Close()
} // }}}

// takes a prefix and a Byte Array and returns a string which contains the prefix + all byte array elements in a readable format
func formatByteArray(prefix string, data []byte) string { // {{{
    out := prefix
    for _, b := range data {
        out += fmt.Sprintf("%02X ", b)
    }
    out += "\n"
    return out
} // }}}

// takes a prefix and a uint32 Array and returns a string which contains the prefix + all uint32 array elements in a readable format
func formatUint32Array(prefix string, data []uint32) string { // {{{
    out := prefix
    for _, u := range data {
        out += fmt.Sprintf("%d ", u)
    }
    out += "\n"
    return out
} // }}}

// takes a prefix and a string Array and returns a string which contains the prefix + all string array elements in a readable format
func formatStringArray(prefix string, data []string) string { // {{{
    out := prefix
    for index, s := range data {
        out += fmt.Sprintf("%i: %s", index, s)
    }
    out += "\n"
    return out
} // }}}
