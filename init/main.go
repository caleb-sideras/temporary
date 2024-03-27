package main

import (
	"io"
	"os"
)

//go:generate go run main.go

func main() {

	inputFile, err := os.Open("../definitions.txt")
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create("../definitions.go")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		panic(err)
	}

	// Default run.go funcs
	runInputFile, err := os.Open("../run2.txt")
	if err != nil {
		panic(err)
	}
	defer runInputFile.Close()

	runOutputFile, err := os.Create("../run2.go")
	if err != nil {
		panic(err)
	}
	defer runOutputFile.Close()

	_, err = io.Copy(runOutputFile, runInputFile)
	if err != nil {
		panic(err)
	}

	// Default Temp struct type
	tempInputFile, err := os.Open("../temp.txt")
	if err != nil {
		panic(err)
	}
	defer tempInputFile.Close()

	tempOnputFile, err := os.Create("../temp.go")
	if err != nil {
		panic(err)
	}
	defer tempOnputFile.Close()

	_, err = io.Copy(tempOnputFile, tempInputFile)
	if err != nil {
		panic(err)
	}

}
