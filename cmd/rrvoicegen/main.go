package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

var (
	version string

	flagCSV  = flag.String("csv", "custom.csv", "Path to CSV file.")
	flagDest = flag.String("dest", "custom", "Dir of where generated files would be stored.")

	flagPollyEngine = flag.String("polly_engine", "standard", "Polly engine (see https://docs.aws.amazon.com/polly/latest/dg/API_DescribeVoices.html)")
	flagPollyLang   = flag.String("polly_lang", "en-US", "Polly language (see https://docs.aws.amazon.com/polly/latest/dg/API_DescribeVoices.html)")
	flagPollyVoice  = flag.String("polly_voice", "Matthew", "Polly voice (see https://docs.aws.amazon.com/polly/latest/dg/voicelist.html)")

	flagVersion = flag.Bool("version", false, "prints version of the application")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println("Version:", version)
		return
	}

	// Read the entire CSV file into memory
	data, err := os.ReadFile(*flagCSV)
	if err != nil {
		fmt.Printf("Failed to read CSV file: %v\n", err)
		return
	}

	// Trim spaces and split into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Connect to AWS
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_DEFAULT_REGION"))},
	)
	if err != nil {
		fmt.Println("Error creating AWS session:", err)
		return
	}

	// Create a Polly client
	svc := polly.New(sess)

	// Throw error if destination directory exists
	if _, err := os.Stat(*flagDest); os.IsExist(err) {
		fmt.Printf("Destination directory already exists: %v\n", err)
		return
	}

	// Create directory
	err = os.Mkdir(*flagDest, 0755)
	if err != nil {
		fmt.Printf("Failed to create destination directory: %v\n", err)
		return
	}

	fmt.Println("Generating and downloading sounds from AWS Polly. Please wait...")

	// Process each line
	wg := &sync.WaitGroup{}
	sem := make(chan struct{}, 20) // Limit to 20 workers

	for _, line := range lines {
		wg.Add(1)
		sem <- struct{}{} // Acquire a token
		go func(line string) {
			defer wg.Done()
			processLine(line, svc, *flagDest)
			<-sem // Release the token
		}(line)
	}
	wg.Wait()

	// Wait for all workers to finish
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}

	fmt.Println("Application completed the task. Bye!")
}

func processLine(line string, svc *polly.Polly, dest string) {
	record, err := csv.NewReader(strings.NewReader(line)).Read()
	if err != nil {
		fmt.Printf("Failed to parse CSV line: %v\n", err)
		fmt.Println("Failed line:", line)
		os.Exit(1)
		return
	}

	filename := record[0]
	voiceLine := record[1]

	// Synthesize speech
	input := &polly.SynthesizeSpeechInput{
		OutputFormat: aws.String("pcm"),
		Text:         aws.String(voiceLine),
		TextType:     aws.String("ssml"),
		SampleRate:   aws.String("16000"),

		Engine:       aws.String(*flagPollyEngine),
		LanguageCode: aws.String(*flagPollyLang),
		VoiceId:      aws.String(*flagPollyVoice),
	}

	// Get output response
	output, err := svc.SynthesizeSpeech(input)
	if err != nil {
		fmt.Printf("Failed to get response from Polly: %v\n", err)
		fmt.Println("Failed src voice-line:", voiceLine)
		os.Exit(1)
		return
	}

	// Read the output
	audioBytes, err := io.ReadAll(output.AudioStream)
	if err != nil {
		fmt.Printf("Failed to download audio from Polly: %v\n", err)
		os.Exit(1)
		return
	}

	// Convert byte data to int for the encoder
	audioData := make([]int, len(audioBytes)/2)
	for i := 0; i < len(audioBytes); i += 2 {
		audioData[i/2] = int(int16(audioBytes[i]) | int16(audioBytes[i+1])<<8)
	}

	// Find the maximum absolute value in the audio data
	maxVal := 0
	for _, val := range audioData {
		absVal := val
		if absVal < 0 {
			absVal = -absVal
		}
		if absVal > maxVal {
			maxVal = absVal
		}
	}

	// Calculate the normalization factor
	normFactor := 32767.0 / float64(maxVal)

	// Normalize the audio data
	for i := range audioData {
		audioData[i] = int(float64(audioData[i]) * normFactor)
	}

	// Create an audio.IntBuffer with your data, mono channel.
	buf := &audio.IntBuffer{Data: audioData, Format: &audio.Format{SampleRate: 16000, NumChannels: 1}}

	// Create a new encoder that will write to a file.
	outFile, err := os.Create(fmt.Sprintf("%s%c%s", *flagDest, os.PathSeparator, filename))
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		os.Exit(1)
		return
	}

	enc := wav.NewEncoder(outFile, buf.Format.SampleRate, 16, buf.Format.NumChannels, 1)

	// Write audio data to the encoder
	if err := enc.Write(buf); err != nil {
		fmt.Printf("Failed to write audio data to file: %v\n", err)
		os.Exit(1)
		return
	}

	// Close the encoder and the underlying file
	if err := enc.Close(); err != nil {
		fmt.Printf("Failed to close encoder and file: %v\n", err)
		os.Exit(1)
		return
	}
}
