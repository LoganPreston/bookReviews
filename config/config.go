package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var Token string

type configStruct struct {
	Key     string `json:"key"`
}

//read a json configuration file and provide access to variables
//currently provides Token and BotPrefix
func ReadConfig() error {
	var (
		config *configStruct
		file   []byte
		err    error
	)
	//Read the configuration file, token and prefix. Handle the err
	if file, err = ioutil.ReadFile("./config.json"); err != nil {
		fmt.Println(err.Error())
		return err
	}

	//unmarshall the file into the main struct. handle err.
	if err = json.Unmarshal(file, &config); err != nil {
		fmt.Println(err.Error())
		return err
	}

	//unpack struct
	Token = config.Token

	//bot info set up successfully at this point, return nothing if success.
	return nil
}
