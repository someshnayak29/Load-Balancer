package cmd

import (
	"bufio"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Algorithm     string `yaml:"algorithm"`
	Port          int    `yaml:"port"`
	Strict        bool   `yaml:"strict"`
	Log           bool   `yaml:"log"`
	XssProtection bool   `yaml:"xss-protection"`
	Servers       []struct {
		Host        string  `yaml:"host"`
		Weight      float64 `yaml:"weight"`
		Connections int     `yaml:"connections"`
	} `yaml:"servers"`
}

// unmarshal config file (yaml) to go struct

func (c *Config) GetConf() *Config {

	yamlFile, err := os.ReadFile("config/config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err  #%v", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

// return a list of all restructed ips mentioned in iplists.txt file

func ReadLines(path string) ([]string, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		ips = append(ips, scanner.Text())
	}

	return ips, scanner.Err()
}
