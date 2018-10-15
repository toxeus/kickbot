package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type bot struct {
	baseURL *url.URL
	chatID  string
}

func (b bot) kick(userID int) error {
	kickURL, _ := b.baseURL.Parse("kickChatMember")
	data := url.Values{
		"chat_id": []string{b.chatID},
		"user_id": []string{strconv.Itoa(userID)},
	}
	resp, err := http.PostForm(kickURL.String(), data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(body)
	return nil
}

func (b bot) deleteMessage(messageID int) error {
	deleteURL, _ := b.baseURL.Parse("deleteMessage")
	data := url.Values{
		"chat_id":    []string{b.chatID},
		"message_id": []string{strconv.Itoa(messageID)},
	}
	resp, err := http.PostForm(deleteURL.String(), data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(body)
	return nil
}

func main() {
	token := flag.String("bot-token", "", "the authentication token for the bot")
	chatID := flag.String("chat-id", "", "the ID of the chat to monitor by the bot")
	userNameLimit := flag.Int("user-name-limit", 30, "the limit for the user name")
	flag.Parse()
	if *token == "" {
		log.Fatal("Error: bot-token needs to be specified")
	}
	if *chatID == "" {
		log.Fatal("Error: chat-id needs to be specified")
	}
	if *userNameLimit < 0 {
		log.Fatal("Error: user-name-limit needs to be non-negative")
	}
	baseURL, err := url.Parse(fmt.Sprintf("https://api.telegram.org/bot%s/", *token))
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	updateURL, _ := baseURL.Parse("getUpdates")
	data := url.Values{
		"allowed_updates": []string{"message"},
		"timeout":         []string{"5"},
	}
	bot := bot{baseURL: baseURL, chatID: *chatID}
	log.Printf("Bot running on chat %s\n", *chatID)
	for {
		resp, err := http.PostForm(updateURL.String(), data)
		if err != nil {
			log.Printf("Error: %s\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		msg := struct {
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  struct {
					MessageID      int `json:"message_id"`
					NewChatMembers []struct {
						ID        int    `json:"id"`
						FirstName string `json:"first_name"`
						LastName  string `json:"last_name"` // optional
						UserName  string `json:"username"`  // optional
					} `json:"new_chat_members"`
				}
			}
		}{}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
			log.Printf("Error: %s\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, update := range msg.Result {
			data.Set("offset", strconv.Itoa(update.UpdateID+1))
			for _, newMember := range update.Message.NewChatMembers {
				if len(newMember.FirstName) > *userNameLimit || len(newMember.LastName) > *userNameLimit {
					log.Printf("Kicking %s %s\n", newMember.FirstName, newMember.UserName)
					bot.kick(newMember.ID)
					bot.deleteMessage(update.Message.MessageID)
				}
			}
		}
	}
}
