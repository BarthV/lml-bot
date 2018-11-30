package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sbstjn/hanu"
)

type interrupt struct {
	EpochDate       int64  `json:"epochDate"`
	DurationMinutes int    `json:"duration"`
	Category        string `json:"category"`
	Fqdn            string `json:"fqdn"`
}

func getCurrentInterrupts() (string, error) {
	now := time.Now()
	filename := fmt.Sprintf("./%v-%s-interrupts.json.log", now.Year(), now.Month().String())

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("Impossible to open JSON log file")
	}
	defer f.Close()

	interruptItem := interrupt{}
	totalTime := time.Duration(0 * time.Second)
	totalEntries := int(0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		json.Unmarshal(scanner.Bytes(), &interruptItem)
		totalEntries = totalEntries + 1
		totalTime = totalTime + time.Duration(interruptItem.DurationMinutes)*time.Minute
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("Impossible to scan JSON log file line per line")
	}

	message := fmt.Sprintf(
		"*Registered interrupt informations for %s :* \n `Total count: %v`\n`Total time : %s`",
		now.Month().String(), totalEntries, totalTime.String(),
	)
	return message, nil
}

func storeInterrupt(d time.Duration, c string, s string) error {
	now := time.Now()
	interruptItem := interrupt{
		EpochDate:       now.Unix(),
		DurationMinutes: int(d.Minutes()),
		Category:        c,
		Fqdn:            s,
	}

	content, err := json.Marshal(interruptItem)
	if err != nil {
		return fmt.Errorf("Impossible to marshal JSON")
	}
	content = append(content, byte('\n'))

	filename := fmt.Sprintf("./%v-%s-interrupts.json.log", now.Year(), now.Month().String())

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Impossible to open JSON log file")
	}
	defer f.Close()

	if _, err = f.Write(content); err != nil {
		return fmt.Errorf("Impossible to append JSON log to interrupt list")
	}

	return nil
}

func main() {
	slack, err := hanu.New(os.Getenv("SLACK_BOT_TOKEN"))

	if err != nil {
		log.Fatal(err)
	}

	Version := "0.1.3"

	slack.Command("add <duration:string> <fqdn:string> (HW|SW|OTHER|UNK)", func(conv hanu.ConversationInterface) {
		durationStr, err := conv.String("duration")
		if err != nil {
			conv.Reply(":warning: *Impossible to parse duration arg.*")
			return
		}
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			conv.Reply(":warning: *Impossible to parse duration*, please use https://golang.org/pkg/time/#ParseDuration format")
			return
		}
		if duration <= time.Duration(0) {
			conv.Reply(":genius:  *I tried to spend negative time at work ...* It didn't worked well")
			return
		}

		fqdn, err := conv.String("fqdn")
		if err != nil {
			conv.Reply(":warning: *Impossible to parse fqdn.* If you cannot specify it please use `nocomment` as a placeholder")
			return
		}

		category, err := conv.Match(2)
		if err != nil {
			conv.Reply(":warning: *Impossible to parse category.*")
			return
		}

		err = storeInterrupt(duration, category, fqdn)
		if err != nil {
			conv.Reply(":skull_and_crossbones: *Impossible to store new interrupt.* Please check this bot health ! " + err.Error())
			return
		}

		slackMsg := fmt.Sprintf(":heavy_check_mark: *New interrupt successfully registered:* `%s` - `%s`", duration.String(), fqdn)
		conv.Reply(slackMsg)
	})

	slack.Command("get_current_month", func(conv hanu.ConversationInterface) {
		message, err := getCurrentInterrupts()
		if err != nil {
			conv.Reply(":warning: *Impossible to read current interrupt log file.* Please check this bot health ! " + err.Error())
			return
		}
		conv.Reply(message)
	})

	slack.Command("version", func(conv hanu.ConversationInterface) {
		conv.Reply("Thanks for asking! I'm running `%s`", Version)
	})

	slack.Listen()
}
