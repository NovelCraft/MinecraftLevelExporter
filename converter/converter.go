package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/NovelCraft/MinecraftLevelExporter/logger"
	"github.com/xeipuuv/gojsonschema"
)

// Section represents a section of a Minecraft world. It contains the x, y, z
// coordinates of the section and a 3D array of blocks.
type Section struct {
	X      int       `json:"x"`
	Y      int       `json:"y"`
	Z      int       `json:"z"`
	Blocks [][][]int `json:"blocks"`
}

// inputJsonSchema is the json schema for the input json file. It must be a 3D
// array of integers.
const inputJsonSchema string = `
{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "array",
	"items": {
		"type": "array",
		"items": {
			"type": "array",
			"items": {
				"type": "integer"
			}
		}
	}
}
`

func main() {
	// Validate arguments.
	if len(os.Args) < 2 {
		logger.Error("no input file specified")
		logger.Info("Usage: %s <input file>", os.Args[0])
		return
	}

	if len(os.Args) > 2 {
		logger.Error("too many arguments")
		logger.Info("Usage: %s <input file>", os.Args[0])
		return
	}

	if !strings.HasSuffix(os.Args[1], ".json") {
		logger.Error("input file must be a json file")
		return
	}

	// Read input file.
	jsonContent, err := os.ReadFile(os.Args[1])
	if err != nil {
		logger.Error("failed to read input file: %s", err.Error())
		return
	}

	// Validate input file.
	schemaLoader := gojsonschema.NewStringLoader(inputJsonSchema)
	documentLoader := gojsonschema.NewBytesLoader((jsonContent))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		logger.Error("failed to validate input file: %s", err.Error())
		return
	}

	if !result.Valid() {
		logger.Error("input file is not valid")
		return
	}

	// Read input JSON to a 3D array.
	var inputArray [][][]int
	err = json.Unmarshal(jsonContent, &inputArray)
	if err != nil {
		logger.Error("failed to parse input file: %s", err.Error())
		return
	}

	// Validate the input array.
	if !validateArray(inputArray) {
		logger.Error("input array is not valid")
		return
	}

	// Pad the input array to be a multiple of 16.
	inputArray = padArray(inputArray, -1)

	// Convert the input array to an array of sections.
	sections := convertToSections(inputArray)

	// Make the level data map
	levelData := make(map[string]interface{})
	levelData["type"] = "level_data"
	levelData["sections"] = sections
	levelData["entities"] = make([]interface{}, 0)
	levelData["players"] = make([]interface{}, 0)

	// Convert the level data map to json.
	levelDataJson, err := json.Marshal(levelData)
	if err != nil {
		logger.Error("failed to convert level data to json: %s", err.Error())
		return
	}

	// Write the level data json to a file.
	fileName := strings.TrimSuffix(os.Args[1], ".json") + ".level.json"
	err = os.WriteFile(fileName, levelDataJson, 0644)
	if err != nil {
		logger.Error("failed to write level data to file: %s", err.Error())
		return
	}

	logger.Info("Successfully converted input file to level data")
}

// validateArray checks if the input array is a 3D cubic array of integers.
func validateArray(inputArray [][][]int) bool {
	for i := 0; i < len(inputArray)-1; i++ {
		if len(inputArray[i]) != len(inputArray[i+1]) {
			return false
		}
	}

	for _, yArray := range inputArray {
		for i := 0; i < len(yArray)-1; i++ {
			if len(yArray[i]) != len(yArray[i+1]) {
				return false
			}
		}
	}

	return true
}

// padArray pads the input array to be a multiple of 16.
func padArray(inputArray [][][]int, paddingValue int) [][][]int {
	// Calculate size of each dimension after padding.
	xSize := (len(inputArray) + 15) / 16 * 16
	ySize := (len(inputArray[0]) + 15) / 16 * 16
	zSize := (len(inputArray[0][0]) + 15) / 16 * 16

	// Create the padded array.
	paddedArray := make([][][]int, xSize)
	for i := range paddedArray {
		paddedArray[i] = make([][]int, ySize)
		for j := range paddedArray[i] {
			paddedArray[i][j] = make([]int, zSize)
			// Fill the padded array with the padding value.
			for k := range paddedArray[i][j] {
				paddedArray[i][j][k] = paddingValue
			}
		}
	}

	// Copy the input array to the padded array.
	for i := 0; i < len(inputArray); i++ {
		for j := 0; j < len(inputArray[i]); j++ {
			copy(paddedArray[i][j], inputArray[i][j])
		}
	}

	return paddedArray
}

// convertToSections converts the input array to an array of sections.
func convertToSections(inputArray [][][]int) []Section {
	// Calculate the number of sections needed.
	xSections := len(inputArray) / 16
	ySections := len(inputArray[0]) / 16
	zSections := len(inputArray[0][0]) / 16

	// Create the sections.
	sections := make([]Section, xSections*ySections*zSections)

	// Fill the sections.
	for i := 0; i < xSections; i++ {
		for k := 0; k < zSections; k++ {
			logger.Info("Reading chunk at (%d, *, %d)", i*16, k*16)
			for j := 0; j < ySections; j++ {
				logger.Info("Creating section at (%d, %d, %d)", i*16, j*16, k*16)

				// Create the blocks array and fill it with the input array.
				blocks := make([][][]int, 16)
				for l := range blocks {
					blocks[l] = make([][]int, 16)
					for m := range blocks[l] {
						blocks[l][m] = make([]int, 16)
						copy(blocks[l][m], inputArray[i*16+l][j*16+m][k*16:(k+1)*16])
					}
				}

				sections[i*ySections*zSections+j*zSections+k] = Section{
					X:      i * 16,
					Y:      j * 16,
					Z:      k * 16,
					Blocks: blocks,
				}
			}
		}
	}

	return sections
}
