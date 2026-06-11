package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type LineNotifyService struct {
	channelToken string
	targetUserID string
}

func NewLineNotifyService(channelToken, targetUserID string) *LineNotifyService {
	return &LineNotifyService{
		channelToken: channelToken,
		targetUserID: targetUserID,
	}
}

func (s *LineNotifyService) SendNotification(message string) error {
	if s.channelToken == "" || s.targetUserID == "" {
		log.Println("[LineMessagingAPI] Token or TargetUserID is empty, skipping notification")
		return nil
	}

	apiURL := "https://api.line.me/v2/bot/message/push"

	payload := map[string]interface{}{
		"to": s.targetUserID,
		"messages": []map[string]interface{}{
			{
				"type": "text",
				"text": message,
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+s.channelToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &result)
		return fmt.Errorf("LINE Messaging API failed with status %d: %v", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("[LineMessagingAPI] Successfully pushed message: %s", message)
	return nil
}
