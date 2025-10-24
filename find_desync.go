package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	"github.com/fatih/color"
	"github.com/gocarina/gocsv"
	"github.com/rodaine/table"
)

type Camera struct {
	Name      string `csv:"name"`
	Uri       string `csv:"uri"`
	Apartment string `csv:"apart"`
}
type Frame struct {
	StreamIndex int    `json:"stream_index"`
	PtsTime     string `json:"pts_time"`
}

type MediaData struct {
	Frames []Frame `json:"frames"`
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

type DriftInfo struct {
	ApartName     string
	CameraHash    string
	Diff          float64
	TotalDurDiff  float64
	DurDiffRate   float64
	VideoDuration float64
	AudioDuration float64
}

func NewDriftInfo(apartName, cameraHash string, diff float64) DriftInfo {
	return DriftInfo{
		ApartName:  apartName,
		CameraHash: cameraHash,
		Diff:       diff,
	}
}

func NewDiffInfo(apartName, cameraHash string, diff float64) DiffInfo {
	return DiffInfo{
		ApartName:  apartName,
		CameraHash: cameraHash,
		Diff:       diff,
	}
}

type Analyzer struct {
	apartDiffs  []DiffInfo
	apartDrifts []DriftInfo
}

func NewAnalyzer() Analyzer {
	return Analyzer{
		apartDiffs: []DiffInfo{},
	}
}

func recordTempFile(url string, length int, align bool) string {
	fmt.Printf("Generate temp file from %s \n", url)
	now := time.Now().UnixNano() / 1000
	microtimeStr := strconv.FormatInt(now, 10)
	hash := md5.Sum([]byte(microtimeStr))
	strLength := strconv.Itoa(length)
	filename := filepath.Join("./temp", fmt.Sprintf("%x", hash)+".mkv")
	fmt.Println(filename)
	rtspOpt := ""
	videoFilter := "null"
	audioFilter := "asetnsamples=320"

	if align {
		audioFilter = audioFilter + ",asetpts=PTS-STARTPTS"
		videoFilter = "setpts=PTS-STARTPTS"
	}

	if strings.Contains(url, "rtsp") {
		rtspOpt = "-rtsp_transport tcp "
	}

	cmdLine := "ffmpeg " + rtspOpt + "-i " + url + "  -c:v libx264 -c:a pcm_mulaw -vf '" + videoFilter + "' -af '" +
		audioFilter + "' -t " + strLength + " " + filename

	fmt.Println(cmdLine)

	cmd := exec.Command("sh", "-c", cmdLine)

	_, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Error command : %v", err)
	}

	return filename
}

func recordTempFileCopy(url string, length int) string {
	fmt.Printf("Generate temp file from %s \n", url)
	now := time.Now().UnixNano() / 1000
	microtimeStr := strconv.FormatInt(now, 10)
	hash := md5.Sum([]byte(microtimeStr))
	strLength := strconv.Itoa(length)
	filename := filepath.Join("./temp", fmt.Sprintf("%x", hash)+".mkv")
	fmt.Println(filename)
	rtspOpt := ""

	if strings.Contains(url, "rtsp") {
		rtspOpt = "-rtsp_transport tcp "
	}

	cmdLine := "ffmpeg " + rtspOpt + "-i " + url + "  -c copy  -t " + strLength + " " + filename

	fmt.Println(cmdLine)

	cmd := exec.Command("sh", "-c", cmdLine)

	_, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Error command : %v", err)
	}

	return filename
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

func (a *Analyzer) Reset() {
	a.apartDiffs = []DiffInfo{}
}

func (a *Analyzer) StartTimeDiff(url string, apart string) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	params := map[string]string{
		"url": url,
	}

	cmdTemplate := `ffprobe -analyzeduration 10M -probesize 10M  {%url} 2>&1`
	cmdLine := fillTemplate(cmdTemplate, params)

	fmt.Printf("\n=== Analyzing Start Times for %s ===\n", apart)
	fmt.Println("Command:")
	fmt.Println(cmdLine)

	cmd := exec.Command("sh", "-c", cmdLine)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error(fmt.Sprintf("Error running ffprobe: %v", err))
		return
	}

	outputStr := string(output)
	fmt.Println("\nFFprobe output:")
	fmt.Println(outputStr)

	// Parse video stream start time
	videoStartRegex := regexp.MustCompile(`Stream #\d+:\d+.*Video.*start\s+([\d.]+)`)
	videoMatches := videoStartRegex.FindStringSubmatch(outputStr)

	// Parse audio stream start time
	audioStartRegex := regexp.MustCompile(`Stream #\d+:\d+.*Audio.*start\s+([\d.]+)`)
	audioMatches := audioStartRegex.FindStringSubmatch(outputStr)

	if len(videoMatches) < 2 {
		logger.Warn("Could not find video stream start time")
		return
	}

	if len(audioMatches) < 2 {
		logger.Warn("Could not find audio stream start time")
		return
	}

	videoStart, err := strconv.ParseFloat(videoMatches[1], 64)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing video start time: %v", err))
		return
	}

	audioStart, err := strconv.ParseFloat(audioMatches[1], 64)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing audio start time: %v", err))
		return
	}

	diff := videoStart - audioStart

	fmt.Printf("\n=== START TIME ANALYSIS ===\n")
	fmt.Printf("Video start time: %.6f seconds\n", videoStart)
	fmt.Printf("Audio start time: %.6f seconds\n", audioStart)
	fmt.Printf("Difference:       %.6f seconds\n", diff)

	diffInfo := NewDiffInfo(apart, url, diff)
	a.apartDiffs = append(a.apartDiffs, diffInfo)

	if math.Abs(diff) > 0.1 {
		color.Red("\nSTART TIME MISMATCH: %.6f seconds", diff)
	} else if math.Abs(diff) > 0.01 {
		color.Yellow("\nSmall start time difference: %.6f seconds", diff)
	} else {
		color.Green("\nStart times are aligned")
	}
}

func (a *Analyzer) PTSDiffDrift(uri string, time int, apart string, direct bool, useTime bool) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	videoPackets := []PacketInfo{}
	audioPackets := []PacketInfo{}

	var sourceFile string

	if !direct {
		sourceFile = recordTempFile(uri, time, false)
	} else {
		sourceFile = uri
	}

	readIntervals := "%+#" + strconv.Itoa(time)
	if useTime {
		readIntervals = "%+" + strconv.Itoa(time)
	}

	params := map[string]string{
		"url":           sourceFile,
		"readIntervals": readIntervals,
	}

	cmdVideoLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  \
	   -i "{%url}" -select_streams v  -show_frames -of csv=p=0 -read_intervals "{%readIntervals}" 2>/dev/null`, params)

	cmdAudioLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  \
		-i "{%url}" -select_streams a  -show_frames -of csv=p=0 -read_intervals "{%readIntervals}" 2>/dev/null`, params)

	fmt.Println("Debug audio cmd:" + cmdAudioLine)
	cmdAudio := exec.Command("sh", "-c", cmdAudioLine)
	outputAudio, erra := cmdAudio.CombinedOutput()

	cmdVideo := exec.Command("sh", "-c", cmdVideoLine)
	outputVideo, errv := cmdVideo.CombinedOutput()

	if erra != nil {
		logger.Error(fmt.Sprintf("Error audio command : %v", erra))
		return
	}

	if errv != nil {
		logger.Error(fmt.Sprintf("Error video command: %v", errv))
		return
	}

	r, _ := regexp.Compile(`(?m)^(?:[\w\.]+,){4}([^,]+),(?:[\w\.]+,){5}([^,]+)`)

	videoMatches := r.FindAllStringSubmatch(string(outputVideo), -1)
	audioMatches := r.FindAllStringSubmatch(string(outputAudio), -1)

	for i, match := range videoMatches {
		ptsTime, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			fmt.Println("Video. Cannot parse pts time")
			continue
		}

		dur, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			fmt.Println("Video. Cannot parse duration")
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
			continue
		}

		dur, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}

		audioPackets = append(audioPackets, PacketInfo{
			number:        i + 1,
			pts_time:      ptsTime,
			duration_time: dur,
		})
	}

	fullPackets := min(len(videoPackets), len(audioPackets))

	if len(videoPackets) > len(audioPackets) {
		color.Yellow("Not enough audio packets. Possible desync")
	}

	if len(videoPackets) < len(audioPackets) {
		color.Yellow("Not enough video packets. Possible desync")
	}

	if fullPackets < 2 {
		color.Yellow("Not enough packets to calculate drift")
		return
	}

	fmt.Printf("\n=== Stream: %s ===\n", uri)
	fmt.Printf("Analyzing %d packet pairs\n", fullPackets)

	// Calculate PTS differences for each packet pair
	audioPtsDiffs := make([]float64, fullPackets)
	audioPtsDiffs[0] = 0
	for i := 1; i < fullPackets; i++ {
		audioPtsDiffs[i] = audioPackets[i].pts_time - audioPackets[i-1].pts_time
	}

	// Display table with drift calculation
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("#", "Video PTS time", "Audio PTS time", "Diff", "Drift from prev")
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	// Sample evenly across the packets
	sampleCount := fullPackets
	step := fullPackets / sampleCount

	var totalDrift float64
	var avgDiff float64
	firstDiff := audioPtsDiffs[0]
	lastDiff := audioPtsDiffs[fullPackets-1]

	for i := 0; i < sampleCount; i++ {
		idx := i * step
		if idx >= fullPackets {
			break
		}

		drift := 0.0
		if idx > 0 {
			prevIdx := (i - 1) * step
			drift = audioPtsDiffs[idx] - audioPtsDiffs[prevIdx]
			totalDrift += drift
		}

		avgDiff += audioPtsDiffs[idx]

		tbl.AddRow(
			audioPackets[idx].number,
			fmt.Sprintf("%.3f", videoPackets[idx].pts_time),
			fmt.Sprintf("%.3f", audioPackets[idx].pts_time),
			fmt.Sprintf("%.3f", audioPtsDiffs[idx]),
			fmt.Sprintf("%.4f", drift),
		)
	}

	tbl.Print()

	avgDiff /= float64(sampleCount)
	totalDriftChange := lastDiff - firstDiff
	driftRate := totalDriftChange / float64(fullPackets-1)

	fmt.Printf("\n=== ANALYSIS ===\n")
	fmt.Printf("First PTS diff:      %.3f seconds\n", firstDiff)
	fmt.Printf("Last PTS diff:       %.3f seconds\n", lastDiff)
	fmt.Printf("Average PTS diff:    %.3f seconds\n", avgDiff)
	fmt.Printf("Total drift change:  %.3f seconds\n", totalDriftChange)
	fmt.Printf("Drift per packet:    %.6f seconds\n", driftRate)

	driftInfo := DiffInfo{
		ApartName:  apart,
		CameraHash: uri,
		Diff:       totalDriftChange,
	}

	a.apartDiffs = append(a.apartDiffs, driftInfo)

	if math.Abs(totalDriftChange) > 0.1 {
		color.Red(" DRIFT DETECTED: %.3f seconds change over %d packets", totalDriftChange, fullPackets)
	} else if math.Abs(avgDiff) > 0.5 {
		color.Yellow("\nFIXED OFFSET: %.3f seconds (no drift)", avgDiff)
	} else {
		color.Green("\nStreams are in sync")
	}

	os.Remove(sourceFile)
}

func (a *Analyzer) CheckPTSDiffDrift() {
	for _, item := range a.apartDiffs {
		fmt.Printf("\nCamera: %s\n", item.ApartName)
		fmt.Printf("  Total PTS diff drift: %.3f seconds\n", item.Diff)

		if math.Abs(item.Diff) > 1 {
			color.Red("DRIFT DETECTED")
		} else {
			color.Green("No significant drift")
		}
	}
}

func (a *Analyzer) TracksDiff(uri string, time int, apart string, direct bool, useTime bool) {

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	videoPackets := []PacketInfo{}
	audioPackets := []PacketInfo{}

	var sourceFile string

	if !direct {
		sourceFile = recordTempFile(uri, time, false)
	} else {
		sourceFile = uri
	}

	readIntervals := "%+#" + strconv.Itoa(time)
	if useTime {
		readIntervals = "%+" + strconv.Itoa(time)
	}

	params := map[string]string{
		"url":           sourceFile,
		"readIntervals": readIntervals,
	}

	cmdVideoLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  -i "{%url}" -select_streams v 
		 -show_frames -of csv=p=0 -read_intervals "{%readIntervals}" 2>/dev/null`, params)

	cmdAudioLine := fillTemplate(`ffprobe -v quiet -analyzeduration 5M -probesize 5M  -i "{%url}" -select_streams a 
	     -show_frames -of csv=p=0 -read_intervals "{%readIntervals}" 2>/dev/null`, params)

	fmt.Println("Video command:")
	fmt.Println(cmdVideoLine)

	fmt.Println("Audio command:")
	fmt.Println(cmdAudioLine)

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
	fmt.Printf("Found %d video packets and %d audio packets\n", len(videoMatches), len(audioMatches))
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
		diff := math.Abs(videoPackets[i].pts_time - audioPackets[i].pts_time)
		diffInfo.Diff += diff
		tbl.AddRow(videoPackets[i].number, videoPackets[i].pts_time, videoPackets[i].duration_time,
			audioPackets[i].pts_time, audioPackets[i].duration_time, diff)
	}

	diffInfo.Diff /= float64(fullPackets)

	a.apartDiffs = append(a.apartDiffs, diffInfo)

	tbl.Print()

	os.Remove(sourceFile)

}

func (a *Analyzer) SimpleDiff(url string, time int, apart string, direct bool) bool {

	var params map[string]string
	if !direct {
		sourceFile := recordTempFile(url, time, true)

		params = map[string]string{
			"url":  sourceFile,
			"time": strconv.Itoa(time),
		}
	} else {
		params = map[string]string{
			"url":  url,
			"time": strconv.Itoa(time),
		}
	}

	cmdVideoTemplate := `ffprobe -v quiet -show_entries frame=stream_index,pts_time -select_streams v:0 -read_intervals "%+1" -of json -i {%url}`
	cmdAudioTemplate := `ffprobe -v quiet -show_entries frame=stream_index,pts_time -select_streams a:0 -read_intervals "%+1" -of json -i {%url}`

	cmdVideoLine := fillTemplate(cmdVideoTemplate, params)
	cmdAudioLine := fillTemplate(cmdAudioTemplate, params)

	fmt.Println("Video command:")
	fmt.Println(cmdVideoLine)

	fmt.Println("Audio command:")
	fmt.Println(cmdAudioLine)

	cmdAudio := exec.Command("sh", "-c", cmdAudioLine)
	outputAudio, erra := cmdAudio.CombinedOutput()

	cmdVideo := exec.Command("sh", "-c", cmdVideoLine)
	outputVideo, errv := cmdVideo.CombinedOutput()

	if erra != nil {
		fmt.Printf("Error audio command : %v", erra)
		fmt.Printf("Audio packets: %s", string(outputAudio))
	}

	if errv != nil {
		fmt.Printf("Error video command: %v", errv)
		fmt.Printf("Video : %s", string(outputVideo))
	}

	var vData MediaData
	var aData MediaData
	videoFirstPts := math.Inf(-1)
	audioFirstPts := math.Inf(-1)

	json.Unmarshal(outputVideo, &vData)
	json.Unmarshal(outputAudio, &aData)

	var err error

	if len(vData.Frames) > 0 {
		videoFirstPts, err = strconv.ParseFloat(vData.Frames[0].PtsTime, 64)
	}

	if err != nil {
		panic(err)
	}

	if len(vData.Frames) > 0 {
		audioFirstPts, err = strconv.ParseFloat(aData.Frames[0].PtsTime, 64)
	}

	if err != nil {
		panic(err)
	}

	fmt.Printf("First video packet: %.2f \n", videoFirstPts)
	fmt.Printf("First audio packet: %.2f \n", audioFirstPts)

	return math.Abs(videoFirstPts-audioFirstPts) <= 1

}

func (a *Analyzer) CheckTrackDesync() {
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

	parser := argparse.NewParser("find_desync", "An attempt to programmatically detect audio/video desynchronization")

	cameras := []*Camera{}

	file := parser.String("f", "file", &argparse.Options{Required: false, Help: "File/stream to analyze"})
	csvFile := parser.String("c", "csv", &argparse.Options{Required: false, Help: "File/stream to analyze"})
	packets := parser.Int("p", "packets", &argparse.Options{Required: false, Help: "Number of packets  to analyze. Mutually exclusive with -t"})
	time := parser.Int("t", "time", &argparse.Options{Required: false, Help: "Time of the input to analyze.  Mutually exclusive with -p"})
	method := parser.String("m", "method", &argparse.Options{Required: false, Help: "Method to analyze: trackdiff, drift, firstpackets, startdiff", Default: "startdiff"})
	rawSubject := parser.Int("d", "direct", &argparse.Options{Required: true, Help: "Analyze directly source, or analyze saved slice of the source", Default: 0})

	err := parser.Parse(os.Args)

	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	if time == nil && packets == nil {
		fmt.Errorf("specify either time or packets")
	}

	if *csvFile != "" {
		fileHandle, err := os.OpenFile(*file, os.O_RDWR, os.ModePerm)

		if err != nil {
			panic(err)
		}

		gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {

			return gocsv.LazyCSVReader(in)
		})

		errHandler := func(err *csv.ParseError) bool {
			fmt.Println(err)
			return true
		}

		if err := gocsv.UnmarshalFileWithErrorHandler(fileHandle, errHandler, &cameras); err != nil {
			panic(err)
		}
	} else {
		cameras = append(cameras, &Camera{*file, *file, ""})
	}

	directMode := false

	if *rawSubject == 1 {
		directMode = true
	}

	useTime := false

	if time != nil {
		useTime = true
	}

	analyzer := NewAnalyzer()
	for _, camera := range cameras {
		switch *method {
		case "trackdiff":
			analyzer.TracksDiff(camera.Uri, *packets, camera.Apartment, directMode, useTime)
			analyzer.CheckTrackDesync()
		case "drift":
			analyzer.PTSDiffDrift(camera.Uri, *packets, camera.Apartment, directMode, useTime)
			analyzer.CheckPTSDiffDrift()
		case "firstpackets":
			fmt.Println("Check in record")
			if analyzer.SimpleDiff(camera.Uri, *packets, camera.Apartment, directMode) {
				color.Green("\nTracks in sync :" + camera.Apartment + " " + camera.Uri)
			} else {
				color.Red("\nTracks are desynced: " + camera.Apartment + " " + camera.Uri)
			}
		case "startdiff":
			fmt.Println("Comparison of the first packets")
			analyzer.StartTimeDiff(camera.Uri, camera.Apartment)
		}
	}
}
