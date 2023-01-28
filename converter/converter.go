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
	X      int   `json:"x"`
	Y      int   `json:"y"`
	Z      int   `json:"z"`
	Blocks []int `json:"blocks"`
}

type Size struct {
	X int
	Y int
	Z int
}

// inputJsonSchema is the json schema for the input json file. It must be a 3D
// array of integers.
const inputJsonSchema string = `
{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"size": {
			"type": "array",
			"items": {
				"type": "integer",
				"minimum": 1
			},
			"minItems": 3,
			"maxItems": 3
		},
		"structure": {
			"type": "object",
			"properties": {
				"block_indices": {
					"type": "array",
					"items": {
						"type": "array",
						"items": {
							"type": "integer"
						},
						"minItems": 1
					},
					"minItems": 1
				},
				"palette": {
					"type": "object",
					"properties": {
						"default": {
							"type": "object",
							"properties": {
								"block_palette": {
									"type": "array",
									"items": {
										"type": "object",
										"properties": {
											"name": {
												"type": "string"
											}
										},
										"required": ["name"]
									},
									"minItems": 1
								}
							},
							"required": ["block_palette"]
						}
					},
					"required": ["default"]
				}
			},
			"required": ["block_indices", "palette"]
		}
	},
	"required": ["size", "structure"]
}
`

const blockDictJsonSchema string = `
{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"patternProperties": {
		"^minecraft:\\w+$": {
			"type": "integer"
		}
	}
}
`

const DefaultBlockId = 0
const OutOfRangeBlockId = -1

func main() {
	// Validate arguments.
	if len(os.Args) < 3 {
		logger.Error("no input file or dictionary specified")
		logger.Info("Usage: %s <input file> <dictionary>", os.Args[0])
		return
	}

	if len(os.Args) > 3 {
		logger.Error("too many arguments")
		logger.Info("Usage: %s <input file> <dictionary>", os.Args[0])
		return
	}

	if !strings.HasSuffix(os.Args[1], ".json") || !strings.HasSuffix(os.Args[2], ".json") {
		logger.Error("input file and dictionary file must be JSON files")
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

	// Read input JSON to a map.
	var inputMap map[string]interface{}
	err = json.Unmarshal(jsonContent, &inputMap)
	if err != nil {
		logger.Error("failed to parse input file: %s", err.Error())
		return
	}

	// Read block dictionary.
	jsonContent, err = os.ReadFile(os.Args[2])
	if err != nil {
		logger.Error("failed to read block dictionary: %s", err.Error())
		return
	}

	// Validate block dictionary.
	schemaLoader = gojsonschema.NewStringLoader(blockDictJsonSchema)
	documentLoader = gojsonschema.NewBytesLoader((jsonContent))

	result, err = gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		logger.Error("failed to validate block dictionary: %s", err.Error())
		return
	}

	if !result.Valid() {
		logger.Error("block dictionary is not valid")
		return
	}

	// Read block dictionary to a map.
	var blockDict map[string]int
	err = json.Unmarshal(jsonContent, &blockDict)
	if err != nil {
		logger.Error("failed to parse block dictionary: %s", err.Error())
		return
	}

	// Validate the input map.
	if !validateInput(inputMap) {
		logger.Error("input map is not valid")
		return
	}

	// Convert the input map to an array of sections.
	sections := convertToSections(inputMap, blockDict, DefaultBlockId, OutOfRangeBlockId)

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

// validateInput checks if the input map is a 3D cubic array of integers.
func validateInput(inputMap map[string]interface{}) bool {
	expectedBlockCount := int(inputMap["size"].([]interface{})[0].(float64) * inputMap["size"].([]interface{})[1].(float64) *
		inputMap["size"].([]interface{})[2].(float64))

	blocks := inputMap["structure"].(map[string]interface{})["block_indices"].([]interface{})[0].([]interface{})

	blockCount := len(blocks)

	// Check if the number of blocks is correct.
	if blockCount != expectedBlockCount {
		return false
	}

	// Get the number of different block types.
	var expectedBlockTypeCount int = 0
	for _, e := range blocks {
		num := int(e.(float64))
		if num > expectedBlockTypeCount {
			expectedBlockTypeCount = num
		}
	}
	expectedBlockTypeCount++ // Add 1 because the block count starts at 0.

	blockTypes := inputMap["structure"].(map[string]interface{})["palette"].(map[string]interface{})["default"].(map[string]interface{})["block_palette"].([]interface{})
	blockTypeCount := len(blockTypes)

	// Check if the number of block types is correct.
	if blockTypeCount != expectedBlockTypeCount {
		return false
	} else {
		return true
	}
}

// convertToSections converts the input map to an array of sections.
func convertToSections(inputMap map[string]interface{}, blockDict map[string]int, defaultBlockId int, outOfRangeBlockId int) []Section {
	size := Size{
		X: int(inputMap["size"].([]interface{})[0].(float64)),
		Y: int(inputMap["size"].([]interface{})[1].(float64)),
		Z: int(inputMap["size"].([]interface{})[2].(float64)),
	}

	rawBlocks := inputMap["structure"].(map[string]interface{})["block_indices"].([]interface{})[0].([]interface{})

	blockIndiceDict := make([]string, 0)
	blockTypes := inputMap["structure"].(map[string]interface{})["palette"].(map[string]interface{})["default"].(map[string]interface{})["block_palette"].([]interface{})
	for _, blockType := range blockTypes {
		blockIndiceDict = append(blockIndiceDict, blockType.(map[string]interface{})["name"].(string))
	}

	blocks := make([]int, len(rawBlocks))
	// Transform the block indices to block ids.
	for i, blockIndice := range rawBlocks {
		blockName := blockIndiceDict[int(blockIndice.(float64))]
		blockId, ok := blockDict[blockName]
		if !ok {
			blockId = defaultBlockId
		}
		blocks[i] = blockId
	}

	// Create the sections.
	sectionCount := Size{
		X: (size.X + 15) / 16,
		Y: (size.Y + 15) / 16,
		Z: (size.Z + 15) / 16,
	}

	sections := make([]Section, 0)
	for x := 0; x < sectionCount.X; x++ {
		for y := 0; y < sectionCount.Y; y++ {
			for z := 0; z < sectionCount.Z; z++ {
				offset := x*16*size.X*size.Y + y*16*size.X + z*16

				sectionBlocks := make([]int, 4096)
				for i := 0; i < 4096; i++ {
					if i+offset >= len(blocks) {
						sectionBlocks[i] = outOfRangeBlockId
					} else {
						sectionBlocks[i] = blocks[i+offset]
					}
				}

				section := Section{
					X:      x * 16,
					Y:      y * 16,
					Z:      z * 16,
					Blocks: sectionBlocks,
				}

				sections = append(sections, section)
			}
		}
	}

	return sections
}
