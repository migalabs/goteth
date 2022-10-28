package utils

import (
	"encoding/json"
	"io/ioutil"
)

func GetValIndexesFromJson(filePath string) ([]uint64, error) {

	var validatorIndex []uint64
	// open file and read all the indexes
	fbytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return validatorIndex, err
	}
	err = json.Unmarshal(fbytes, &validatorIndex)

	if err != nil {
		log.Errorf("Error unmarshalling val list: %s", err.Error())
	}

	log.Infof("Readed %d validators", len(validatorIndex))

	return validatorIndex, nil

}

func BoolToUint(input []bool) []uint64 {
	result := make([]uint64, len(input))

	for i, item := range input {
		if item {
			result[i] += 1
		}
	}
	return result
}
