package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

func main() {
	args := os.Args[1:]
	errPipe := os.Stderr

	for _, file := range args {
		if err := download(errPipe, file); err != nil {
			fmt.Println(err)
		}
	}
}

func download(errPipe io.Writer, url string) error {
	var out io.Writer
	var outFile *os.File
	var length int64

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New(response.Status)
	}

	filename := response.Request.URL.Path
	if cd := response.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			filename = params["filename"]
		}
	}
	filename = filepath.Base(path.Clean("/" + filename))

	contentLength := response.Header.Get("Content-Length")
	if contentLength != "" {
		length, err = strconv.ParseInt(contentLength, 10, 32)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("`%v` - could not find file length using HEAD request", filename)
	}

	outFile, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()

	out = outFile
	buf := make([]byte, 4068)
	total := int64(0)
	i := 0

	for {
		n, err := response.Body.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		total += int64(n)
		if _, err := out.Write(buf[:n]); err != nil {
			return err
		}
		i++
		drawProgress(errPipe, total, length)
	}

	drawProgress(errPipe, total, length)
	fmt.Fprintf(errPipe, "\n `%v` has been successfully downloaded [%v]\n", filename, byteUnitStr(total))

	return nil
}

func progress(percents int64) string {
	equalses := percents * 38 / 100
	if equalses < 0 {
		equalses = 0
	}

	spaces := 38 - equalses
	if spaces < 0 {
		spaces = 0
	}

	return strings.Repeat("=", int(equalses)) + ">" + strings.Repeat(" ", int(spaces))
}

func drawProgress(errPipe io.Writer, total, length int64) {
	percents := (100 * total) / length
	progress := progress(percents)
	size := byteUnitStr(total)

	if length < 1 {
		fmt.Fprintf(errPipe, "\r     [ <=>                                  ] %d\t            ", total)
	} else {
		fmt.Fprintf(errPipe, "\r%3d%% [%s] %s\t            ", percents, progress, size)
	}
}

func byteUnitStr(n int64) string {
	var unit string
	size := float64(n)

	for i := 1; i < len(byteUnits); i++ {
		if size < 1000 {
			unit = byteUnits[i-1]
			break
		}
		size = size / 1000
	}

	return fmt.Sprintf("%.3g %s", size, unit)
}
