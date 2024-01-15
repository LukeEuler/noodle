package noodle

import (
	"bytes"
	"os"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/LukeEuler/dolly/log"
	"github.com/pkg/errors"
)

var Conf = new(Config)

func LoadConfig(configPath string) {
	_, err := toml.DecodeFile(configPath, Conf)
	if err != nil {
		log.Entry.Fatal(err)
	}

	err = log.AddFileOut(Conf.Log.File.Path, 5, Conf.Log.File.MaxAge)
	if err != nil {
		log.Entry.WithError(err).Fatal(err)
	}

	if Conf.NodeCheck.MaxRecordNum < 1 {
		Conf.NodeCheck.MaxRecordNum = 1
	}
	err = writable(Conf.NodeCheck.File)
	if err != nil {
		log.Entry.WithError(err).Fatal(err)
	}
	Conf.NodeRecord = new(Beans)
	_, err = toml.DecodeFile(Conf.NodeCheck.File, &Conf.NodeRecord)
	if err != nil {
		log.Entry.Fatal(err)
	}
}

type Config struct {
	Name      string `toml:"name"`
	NodeCheck struct {
		CheckInterval  int64             `toml:"check_interval_s"`
		File           string            `toml:"file"`
		MaxRecordNum   int               `toml:"max_record_num"`
		Method         string            `toml:"method"`
		URL            string            `toml:"url"`
		Body           string            `toml:"body"`
		HeightJSONPath string            `toml:"height_json_path"`
		Headers        map[string]string `toml:"headers"`
	} `toml:"node_check"`
	NodeRecord *Beans `toml:"-"`
	Conamds    struct {
		Content [][]string `toml:"content"`
	} `toml:"commands"`
	Ding struct {
		Enable  bool     `toml:"enable"`
		URL     string   `toml:"url"`
		Secret  string   `toml:"secret"`
		Mobiles []string `toml:"mobiles"`
	} `toml:"ding"`
	Lark struct {
		Enable bool   `toml:"enable"`
		URL    string `toml:"url"`
		Secret string `toml:"secret"`
	} `toml:"lark"`
	Log struct {
		File struct {
			Path   string `toml:"path"`
			MaxAge int    `toml:"max_age"`
		} `toml:"file"`
	} `toml:"log"`
}

func (c *Config) SaveNodeRecord() error {
	buf := bytes.NewBuffer([]byte{})
	encoder := toml.NewEncoder(buf)
	err := encoder.Encode(c.NodeRecord)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(Conf.NodeCheck.File, buf.Bytes(), 0644)
	return errors.WithStack(err)
}

type Beans struct {
	Bean []Bean `toml:"bean"`
}

type Bean struct {
	Height    string
	Timestamp int64
	Time      string
}

func writable(filePath string) error {
	_, err := os.Stat(filePath)
	if err != nil {
		_, err = os.Create(filePath)
		return errors.WithStack(err)
	}
	return errors.WithStack(syscall.Access(filePath, syscall.O_RDWR))
}
