package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gocarina/gocsv"
	"github.com/rodaine/table"
)

type Camera struct {
	Name      string `csv:"name"`
	Uri       string `csv:"uri"`
	Apartment string `csv:"apart"`
}

func fillTemplate(content string, params map[string]string) string {

	var cmdTemplate string = ""

	lines := strings.Split(string(content), "\n")
	linesWithoutComments := []string{}
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "#") {
			linesWithoutComments = append(linesWithoutComments, lines[i])
		}
	}
	cmdTemplate = strings.Join(linesWithoutComments, "\n")

	for k, v := range params {
		cmdTemplate = strings.ReplaceAll(cmdTemplate, "{%"+k+"}", v)
	}

	return cmdTemplate
}

type PacketInfo struct {
	number        int
	pts_time      float64
	duration_time float64
}

type DiffInfo struct {
	ApartName  string
	CameraHash string
	Diff       float64
}

func NewDiffInfo(apartName, cameraHash string, diff float64) DiffInfo {
	return DiffInfo{
		ApartName:  apartName,
		CameraHash: cameraHash,
		Diff:       diff,
	}
}

type Analyzer struct {
	apartDiffs []DiffInfo
}

func NewAnalyzer() Analyzer {
	return Analyzer{
		apartDiffs: []DiffInfo{},
	}
}

func (a *Analyzer) Reset() {
	a.apartDiffs = []DiffInfo{}
}

func (a *Analyzer) AvCompare(uri string, packets int, apart string) {

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	videoPackets := []PacketInfo{}
	audioPackets := []PacketInfo{}

	params := map[string]string{
		"url":     uri,
		"packets": strconv.Itoa(packets),
	}

	cmdVideoLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  \
	   -i "{%url}" -select_streams v  -show_frames -of csv=p=0 -read_intervals "%+#{%packets}" 2>/dev/null`, params)

	cmdAudioLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  \
		-i "{%url}" -select_streams a  -show_frames -of csv=p=0 -read_intervals "%+#{%packets}" 2>/dev/null`, params)

	//	fmt.Println("Video command:")
	//	fmt.Println(cmdVideoLine)

	//	fmt.Println("Audio command:")
	//	fmt.Println(cmdAudioLine)

	cmdAudio := exec.Command("sh", "-c", cmdAudioLine)
	outputAudio, erra := cmdAudio.CombinedOutput()

	cmdVideo := exec.Command("sh", "-c", cmdVideoLine)
	outputVideo, errv := cmdVideo.CombinedOutput()

	if erra != nil {
		logger.Error(fmt.Sprintf("Error audio command : %v", erra))
		logger.Error(fmt.Sprintf("Audio packets: %s", string(outputAudio)))
	}

	if errv != nil {
		logger.Error(fmt.Sprintf("Error video command: %v", errv))
		logger.Error(fmt.Sprintf("Video : %s", string(outputVideo)))
	}

	r, _ := regexp.Compile(`(?m)^(?:[\w\.]+,){4}([^,]+),(?:[\w\.]+,){5}([^,]+)`)

	videoMatches := r.FindAllStringSubmatch(string(outputVideo), -1)
	audioMatches := r.FindAllStringSubmatch(string(outputAudio), -1)

	for i, match := range videoMatches {
		ptsTime, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			fmt.Printf("Cannot parse pts_time: %v", err)
			continue
		}

		dur, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			fmt.Printf("Cannot parse duration: %v", err)
			continue
		}

		videoPackets = append(videoPackets, PacketInfo{
			number:        i + 1,
			pts_time:      ptsTime,
			duration_time: dur,
		})
	}

	for i, match := range audioMatches {
		ptsTime, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			fmt.Printf("Cannot parse pts_time: %v", err)
			continue
		}

		dur, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			fmt.Printf("Cannot parse duration: %v", err)
			continue
		}

		audioPackets = append(audioPackets, PacketInfo{
			number:        i + 1,
			pts_time:      ptsTime,
			duration_time: dur,
		})
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("#", "Video PTS time", "Video duration", "Audio PTS time", "Audio duration", "diff")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	fullPackets := min(len(videoPackets), len(audioPackets))

	diffInfo := NewDiffInfo(apart, uri, 0)

	for i := 0; i < fullPackets; i++ {
		diff := videoPackets[i].pts_time - audioPackets[i].pts_time
		diffInfo.Diff += diff
		tbl.AddRow(videoPackets[i].number, videoPackets[i].pts_time, videoPackets[i].duration_time,
			audioPackets[i].pts_time, audioPackets[i].duration_time, diff)
	}

	diffInfo.Diff /= float64(fullPackets)

	a.apartDiffs = append(a.apartDiffs, diffInfo)

	//tbl.Print()

}

func (a *Analyzer) CheckDesync() {
	for _, item := range a.apartDiffs {
		if item.Diff > 0.5 {
			fmt.Println("Desyncronization spotted in " + item.ApartName + " " + item.CameraHash)
			fmt.Printf("Average desync: %.2f \n", item.Diff)
		} else {
			fmt.Println(item.CameraHash + " should be in sync ")
		}
	}
}

func main() {

	file := os.Args[1]
	packets, err := strconv.Atoi(os.Args[2])

	if err != nil {
		panic(err)
	}

	fileHandle, err := os.OpenFile(file, os.O_RDWR, os.ModePerm)

	if err != nil {
		panic(err)
	}

	cameras := []*Camera{}

	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {

		return gocsv.LazyCSVReader(in) // Allows use of quotes in CSV
	})

	errHandler := func(err *csv.ParseError) bool {
		fmt.Println(err)
		return true
	}

	if err := gocsv.UnmarshalFileWithErrorHandler(fileHandle, errHandler, &cameras); err != nil {
		panic(err)
	}

	analyzer := NewAnalyzer()
	for _, camera := range cameras {
		analyzer.AvCompare(camera.Uri, packets, camera.Apartment)
		analyzer.CheckDesync()
	}
	//Analyzer{}.AvCompare(uri, packets)
}
