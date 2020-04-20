package config

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"сonnect-companion/logger"

	"gopkg.in/yaml.v2"
)

type Parser interface {
	ParseYAML([]byte) error
}

func (c *Conf) ParseYAML(b []byte) error {
	return yaml.Unmarshal(b, &c)
}

func configLoad(configFile string, p Parser) {
	var err error

	logger.Debug("Load configuration at ")

	if configFile, err = filepath.Abs(configFile); err != nil {
		log.Fatalln(err)
	}

	log.Printf("%+v", configFile)

	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(configFile); err != nil {
		log.Fatalln(err)
	}

	// Read the config file
	yamlBytes, err := ioutil.ReadAll(input)
	input.Close()

	if err != nil {
		log.Fatalln(err)
	}

	// Parse the config
	if err := p.ParseYAML(yamlBytes); err != nil {
		//log.Fatalf("Content: %v", yamlBytes)
		log.Fatalf("Could not parse %q: %v", configFile, err)
	}
}

func GetConfig(configPath string, cnf *Conf) {
	configLoad(configPath, cnf)
}
