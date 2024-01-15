package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	dc "github.com/LukeEuler/dolly/common"
	"github.com/LukeEuler/dolly/log"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/LukeEuler/noodle"
	"github.com/LukeEuler/noodle/consumer"
)

var caller = new(consumer.Consumer)

func main() {
	log.AddConsoleOut(5)
	showVersion := flag.Bool("v", false, "show version")
	configFile := flag.String("c", "config.toml", "set the config file path")
	flag.Parse()

	if *showVersion {
		dc.ShowVersion()
		return
	}

	noodle.LoadConfig(*configFile)
	if noodle.Conf.Ding.Enable {
		caller.SetDingTalk(noodle.Conf.Ding.URL, noodle.Conf.Ding.Secret, noodle.Conf.Ding.Mobiles)
	}
	if noodle.Conf.Lark.Enable {
		caller.SetLark(noodle.Conf.Lark.URL, noodle.Conf.Lark.Secret)
	}

	height, err := getNowHeight()
	if err != nil {
		log.Entry.WithError(err).Fatal(err)
	}

	restart, err := updateRecord(height)
	if err != nil {
		log.Entry.WithError(err).Fatal(err)
	}

	if restart {
		err = doRestart()
		if err != nil {
			log.Entry.WithError(err).Fatal(err)
		}
	}
}

func getNowHeight() (string, error) {
	body := strings.NewReader(noodle.Conf.NodeCheck.Body)

	request, err := http.NewRequest(noodle.Conf.NodeCheck.Method, noodle.Conf.NodeCheck.URL, body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	for k, v := range noodle.Conf.NodeCheck.Headers {
		request.Header.Add(k, v)
	}

	command, _ := dc.GetCurlCommand(request)
	log.Entry.Debug(command)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", errors.WithStack(err)
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	value := gjson.Get(string(result), noodle.Conf.NodeCheck.HeightJSONPath)
	return value.String(), nil
}

func updateRecord(height string) (bool, error) {
	now := time.Now()
	newItme := noodle.Bean{
		Height:    height,
		Timestamp: now.Unix(),
		Time:      now.String(),
	}
	if len(noodle.Conf.NodeRecord.Bean) == 0 {
		noodle.Conf.NodeRecord.Bean = append(noodle.Conf.NodeRecord.Bean, newItme)
		err := noodle.Conf.SaveNodeRecord()
		return false, err
	}

	sort.SliceStable(noodle.Conf.NodeRecord.Bean, func(i, j int) bool {
		return noodle.Conf.NodeRecord.Bean[i].Timestamp < noodle.Conf.NodeRecord.Bean[j].Timestamp
	})
	lastItem := noodle.Conf.NodeRecord.Bean[len(noodle.Conf.NodeRecord.Bean)-1]
	if lastItem.Height == height {
		duration := time.Duration(noodle.Conf.NodeCheck.CheckInterval) * time.Second
		lastTime := time.Unix(lastItem.Timestamp, 0)
		delay := now.Sub(lastTime)
		if delay >= duration {
			err := caller.Send("node : "+noodle.Conf.Name,
				"yellow",
				fmt.Sprintf("stopped for %s >= %s",
					delay, duration),
				false)
			if err != nil {
				log.Entry.WithError(err).Error(err)
			}
			return true, nil
		}
		return false, nil
	}

	noodle.Conf.NodeRecord.Bean = append(noodle.Conf.NodeRecord.Bean, newItme)
	length := len(noodle.Conf.NodeRecord.Bean)
	if length > noodle.Conf.NodeCheck.MaxRecordNum {
		noodle.Conf.NodeRecord.Bean = noodle.Conf.NodeRecord.Bean[length-noodle.Conf.NodeCheck.MaxRecordNum:]
	}
	err := noodle.Conf.SaveNodeRecord()
	return false, err
}

func doRestart() error {
	buf := bytes.NewBufferString("")
	for _, list := range noodle.Conf.Conamds.Content {
		buf.WriteString(strings.Join(list, " ") + "\n")
	}
	err := caller.Send("restart : "+noodle.Conf.Name, "yellow", buf.String(), false)
	if err != nil {
		log.Entry.WithError(err).Error(err)
	}

	for i, list := range noodle.Conf.Conamds.Content {
		log.Entry.Infof("connamd %d: %v", i, list)
		bs, err := exec.Command(list[0], list[1:]...).Output()
		log.Entry.Info(string(bs))
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
